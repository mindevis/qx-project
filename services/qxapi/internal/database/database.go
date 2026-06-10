package database

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func Connect(dsn string) (*gorm.DB, error) {
	return Open(mysql.Open(dsn))
}

var openGORM = gorm.Open

// Open initializes GORM with the given dialector (used in tests with SQLite).
func Open(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := openGORM(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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

var migrateUsers = func(db *gorm.DB) error {
	dropRedundantUsersEmailIndex(db)
	return db.AutoMigrate(&models.User{})
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
