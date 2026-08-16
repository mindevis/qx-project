package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := Open(mysql.Open(dsn))
	if err != nil {
		return nil, wrapConnectError(err)
	}
	return db, nil
}

var openGORM = gorm.Open

// Open initializes GORM with the given dialector (used in tests with SQLite).
func Open(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := openGORM(dialector, &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		}),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := configurePool(db); err != nil {
		return nil, err
	}

	if err := migrateUsers(db); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	return db, nil
}

const schemaUnicodeCI = "utf8mb4_unicode_ci"

// monitoringCollationTables matches docs/migrations/2026-06-30_game_servers_collation.sql.
var monitoringCollationTables = []string{
	"game_servers",
	"game_server_monitoring_feedback",
}

var migrateUsers = func(db *gorm.DB) error {
	dropRedundantUsersEmailIndex(db)
	if err := db.AutoMigrate(
		&models.User{},
		&models.LauncherDevice{},
		&models.LauncherInstance{},
		&models.OfflineProfile{},
		&models.MojangLink{},
		&models.LaunchRequest{},
		&models.ModInstallRequest{},
		&models.PrepareRequest{},
		&models.ModUninstallRequest{},
		&models.InstanceFileRequest{},
		&models.InstanceResourceUploadRequest{},
		&models.InstanceResourceExportRequest{},
		&models.UserCosmetics{},
		&models.Server{},
		&models.SSHCredential{},
		&models.Agent{},
		&models.GameServer{},
		&models.GameServerMonitoringFeedback{},
		&models.GameServerInstanceBinding{},
	); err != nil {
		return err
	}
	fixMonitoringTablesCollation(db)
	widenModListColumns(db)
	dropResourceBlobColumns(db)
	return nil
}

// modListColumns are columns that hold JSON lists of per-mod metadata and can
// exceed MySQL's 64 KB TEXT limit for a large modpack.
var modListColumns = map[string][]string{
	"launcher_instances": {"mods", "resource_packs", "shaders", "datapacks"},
	"game_servers": {
		"monitoring_mods_json", "monitoring_client_mods_json",
		"monitoring_resourcepacks_json", "monitoring_client_resourcepacks_json",
		"monitoring_shaders_json", "monitoring_client_shaders_json",
		"monitoring_plugins_json",
	},
	"game_server_instance_bindings": {"client_mod_enabled", "client_resourcepack_enabled", "client_shader_enabled"},
}

// widenModListColumns upgrades mod/resource list columns from TEXT (64 KB) to
// MEDIUMTEXT (16 MB) on MySQL. A large modpack's mod metadata exceeds the TEXT
// limit, which fails the write and blocks any further install. Applied
// explicitly (and idempotently) so existing databases are guaranteed widened
// even if AutoMigrate does not detect the text-subtype change.
func widenModListColumns(db *gorm.DB) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "mysql" {
		return
	}
	m := db.Migrator()
	for table, columns := range modListColumns {
		if !m.HasTable(table) {
			continue
		}
		for _, col := range columns {
			var dataType string
			if err := db.Raw(
				`SELECT DATA_TYPE FROM information_schema.COLUMNS
				 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
				table, col,
			).Scan(&dataType).Error; err != nil {
				continue
			}
			if dataType == "" || dataType == "mediumtext" || dataType == "longtext" {
				continue
			}
			if err := db.Exec("ALTER TABLE " + table + " MODIFY COLUMN " + col + " MEDIUMTEXT").Error; err != nil {
				log.Printf("warning: widen %s.%s to mediumtext: %v", table, col, err)
			}
		}
	}
}

// resourceBlobTables used to store jar/zip bytes as LONGTEXT. Files live in
// MinIO now; leftover columns (and old rows) still crush InnoDB until dropped.
var resourceBlobTables = []string{
	"instance_resource_upload_requests",
	"instance_resource_export_requests",
}

func dropResourceBlobColumns(db *gorm.DB) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "mysql" {
		return
	}
	m := db.Migrator()
	for _, table := range resourceBlobTables {
		if !m.HasTable(table) {
			continue
		}
		var dataType string
		if err := db.Raw(
			`SELECT DATA_TYPE FROM information_schema.COLUMNS
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = 'content_b64'`,
			table,
		).Scan(&dataType).Error; err != nil || dataType == "" {
			continue
		}
		if err := db.Exec("ALTER TABLE `" + table + "` DROP COLUMN content_b64").Error; err != nil {
			log.Printf("warning: drop %s.content_b64: %v", table, err)
		}
	}
}

// fixMonitoringTablesCollation aligns monitoring tables on MySQL.
// GORM AutoMigrate uses utf8mb4_0900_ai_ci by default; monitoring JOINs need utf8mb4_unicode_ci.
func fixMonitoringTablesCollation(db *gorm.DB) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "mysql" {
		return
	}
	m := db.Migrator()
	for _, table := range monitoringCollationTables {
		if !m.HasTable(table) {
			continue
		}
		var collation string
		if err := db.Raw(
			`SELECT TABLE_COLLATION FROM information_schema.TABLES
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`,
			table,
		).Scan(&collation).Error; err != nil || collation == "" || collation == schemaUnicodeCI {
			continue
		}
		if err := db.Exec(
			"ALTER TABLE " + table + " CONVERT TO CHARACTER SET utf8mb4 COLLATE " + schemaUnicodeCI,
		).Error; err != nil {
			log.Printf("warning: fix %s collation (%s -> %s): %v", table, collation, schemaUnicodeCI, err)
		}
	}
}

// dropRedundantUsersEmailIndex removes a legacy non-unique email index
// that collided with GORM AutoMigrate (email is already UNIQUE on the column).
func dropRedundantUsersEmailIndex(db *gorm.DB) {
	m := db.Migrator()
	if !m.HasTable(&models.User{}) {
		return
	}
	if m.HasIndex(&models.User{}, "idx_users_email") {
		_ = m.DropIndex(&models.User{}, "idx_users_email")
	}
}

var configurePool = func(db *gorm.DB) error {
	sqlDB, err := poolDB(db)
	if err != nil {
		return fmt.Errorf("sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return nil
}

var poolDB = func(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

func Ping(db *gorm.DB) error {
	sqlDB, err := poolDB(db)
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
