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
	// ADD COLUMN only — never DROP/MODIFY/CONVERT here. AutoMigrate must not
	// add MEDIUMTEXT/VARCHAR NOT NULL without a default on populated MySQL tables
	// or qxapi never reaches Listen.
	ensureSchemaAdditions(db)
	fixMonitoringTablesCollation(db)
	widenModListColumns(db)
	return nil
}

type schemaAddition struct {
	table  string
	column string
	mysql  string
	sqlite string
}

// Columns that must not go through GORM AutoMigrate ADD (NOT NULL TEXT/MEDIUMTEXT
// or NOT NULL VARCHAR without DEFAULT fails on existing MySQL rows).
var schemaAdditions = []schemaAddition{
	{
		table:  "game_servers",
		column: "content_resources",
		mysql:  "`content_resources` MEDIUMTEXT NULL",
		sqlite: "content_resources TEXT",
	},
	{
		table:  "game_servers",
		column: "min_memory_mb",
		mysql:  "`min_memory_mb` INT NULL",
		sqlite: "min_memory_mb INTEGER",
	},
	{
		table:  "game_servers",
		column: "max_memory_mb",
		mysql:  "`max_memory_mb` INT NULL",
		sqlite: "max_memory_mb INTEGER",
	},
	{
		table:  "game_servers",
		column: "extra_jvm_args",
		mysql:  "`extra_jvm_args` TEXT NULL",
		sqlite: "extra_jvm_args TEXT",
	},
	{
		table:  "game_servers",
		column: "extra_args",
		mysql:  "`extra_args` TEXT NULL",
		sqlite: "extra_args TEXT",
	},
	{
		table:  "prepare_requests",
		column: "progress_message",
		mysql:  "`progress_message` VARCHAR(256) NOT NULL DEFAULT ''",
		sqlite: "progress_message TEXT NOT NULL DEFAULT ''",
	},
	{
		table:  "launch_requests",
		column: "progress_message",
		mysql:  "`progress_message` VARCHAR(256) NOT NULL DEFAULT ''",
		sqlite: "progress_message TEXT NOT NULL DEFAULT ''",
	},
	{
		table:  "launcher_instances",
		column: "managed_by_game_server_id",
		mysql:  "`managed_by_game_server_id` CHAR(36) NULL",
		sqlite: "managed_by_game_server_id TEXT",
	},
}

func ensureSchemaAdditions(db *gorm.DB) {
	if db == nil || db.Dialector == nil {
		return
	}
	mysql := db.Dialector.Name() == "mysql"
	m := db.Migrator()
	for _, col := range schemaAdditions {
		if !m.HasTable(col.table) || columnExists(db, col.table, col.column) {
			continue
		}
		spec := col.sqlite
		quoted := col.table
		if mysql {
			spec = col.mysql
			quoted = "`" + col.table + "`"
		}
		if err := db.Exec("ALTER TABLE " + quoted + " ADD COLUMN " + spec).Error; err != nil {
			log.Printf("warning: add %s.%s: %v", col.table, col.column, err)
		}
	}
	ensureManagedInstanceIndex(db, mysql)
}

func columnExists(db *gorm.DB, table, column string) bool {
	if db.Dialector.Name() == "mysql" {
		var n int64
		if err := db.Raw(
			`SELECT COUNT(*) FROM information_schema.COLUMNS
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
			table, column,
		).Scan(&n).Error; err != nil {
			return false
		}
		return n > 0
	}
	type pragmaCol struct {
		Name string `gorm:"column:name"`
	}
	var cols []pragmaCol
	if err := db.Raw("PRAGMA table_info(" + table + ")").Scan(&cols).Error; err != nil {
		return false
	}
	for _, col := range cols {
		if col.Name == column {
			return true
		}
	}
	return false
}

func ensureManagedInstanceIndex(db *gorm.DB, mysql bool) {
	m := db.Migrator()
	if !m.HasTable("launcher_instances") || !columnExists(db, "launcher_instances", "managed_by_game_server_id") {
		return
	}
	if mysql {
		var n int64
		if err := db.Raw(
			`SELECT COUNT(*) FROM information_schema.STATISTICS
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`,
			"launcher_instances", "idx_instances_managed_server",
		).Scan(&n).Error; err != nil || n > 0 {
			return
		}
		if err := db.Exec(
			"CREATE INDEX `idx_instances_managed_server` ON `launcher_instances` (`managed_by_game_server_id`)",
		).Error; err != nil {
			log.Printf("warning: create idx_instances_managed_server: %v", err)
		}
		return
	}
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_instances_managed_server ON launcher_instances (managed_by_game_server_id)",
	).Error; err != nil {
		log.Printf("warning: create idx_instances_managed_server: %v", err)
	}
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
// MinIO now. Do not DROP these columns during API boot — ALTER rebuilds the
// table and can keep qxapi from listening. Use docs/migrations/2026-08-16_drop_resource_content_b64.sql.
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
