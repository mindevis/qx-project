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
		&models.GameServerNetwork{},
		&models.GameServerNetworkMember{},
		&models.ServerOllama{},
	); err != nil {
		return err
	}
	// These fields use gorm:"-:migration" so AutoMigrate will not create them
	// (MySQL ADD MEDIUMTEXT/VARCHAR NOT NULL on live tables takes the API down).
	for _, stmt := range []string{
		"ALTER TABLE game_servers ADD COLUMN content_resources TEXT",
		"ALTER TABLE game_servers ADD COLUMN min_memory_mb INTEGER",
		"ALTER TABLE game_servers ADD COLUMN max_memory_mb INTEGER",
		"ALTER TABLE game_servers ADD COLUMN extra_jvm_args TEXT",
		"ALTER TABLE game_servers ADD COLUMN extra_args TEXT",
		"ALTER TABLE prepare_requests ADD COLUMN progress_message TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE launch_requests ADD COLUMN progress_message TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE launcher_instances ADD COLUMN managed_by_game_server_id TEXT",
		"CREATE INDEX IF NOT EXISTS idx_instances_managed_server ON launcher_instances (managed_by_game_server_id)",
	} {
		_ = db.Exec(stmt).Error
	}
	return nil
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
