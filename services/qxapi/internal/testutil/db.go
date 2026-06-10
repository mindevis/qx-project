package testutil

import (
	"database/sql"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var sqliteOpen = sqlite.Open

var openGORM = gorm.Open

func openSQLiteDB(name string) (*gorm.DB, error) {
	db, err := openGORM(sqliteOpen("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := autoMigrateUsers(db); err != nil {
		return nil, err
	}
	return db, nil
}

func OpenSQLiteDB(t testing.TB) *gorm.DB {
	t.Helper()
	db, err := openSQLiteDB(t.Name())
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

var autoMigrateUsers = func(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{})
}

func CloseDB(t testing.TB, db *gorm.DB) {
	t.Helper()
	if db == nil {
		return
	}
	if err := closeSQL(db); err != nil {
		return
	}
}

var gormSQLDB = func(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

var closeSQL = func(db *gorm.DB) error {
	sqlDB, err := gormSQLDB(db)
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
