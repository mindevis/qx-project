package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var sqliteOpen = sqlite.Open

var openGORM = gorm.Open

func openSQLiteDB(name string) (*gorm.DB, error) {
	// Unique in-memory DB per call — cache=shared so pool connections share schema/data.
	db, err := openGORM(sqliteOpen("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := autoMigrateUsers(db); err != nil {
		return nil, err
	}
	return db, nil
}

func uniqueDBName(t testing.TB) string {
	t.Helper()
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return t.Name() + "_" + hex.EncodeToString(suffix)
}

// MemoryDSN returns a unique in-memory SQLite DSN for tests (-count / parallel safe).
// cache=shared is required so GORM pool connections see the same in-memory database.
func MemoryDSN(t testing.TB) string {
	t.Helper()
	return "file:" + uniqueDBName(t) + "?mode=memory&cache=shared"
}

func OpenSQLiteDB(t testing.TB) *gorm.DB {
	t.Helper()
	db, err := openSQLiteDB(uniqueDBName(t))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { CloseDB(t, db) })
	return db
}

var autoMigrateUsers = func(db *gorm.DB) error {
	return db.AutoMigrate(
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
	)
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
