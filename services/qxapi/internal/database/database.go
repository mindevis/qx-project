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
	return nil
}

// fixMonitoringTablesCollation aligns GORM-created tables with docs/schema.sql on MySQL.
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

// dropRedundantUsersEmailIndex removes a legacy non-unique index from docs/schema.sql
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
