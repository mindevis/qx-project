package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestOpenGORMError(t *testing.T) {
	old := openGORM
	openGORM = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}
	t.Cleanup(func() { openGORM = old })

	if _, err := Open(sqlite.Open("file:openfail?mode=memory&cache=shared")); err == nil {
		t.Fatal("expected open error")
	}
}

func TestConnectInvalidDSN(t *testing.T) {
	if _, err := Connect("not-a-valid-dsn"); err == nil {
		t.Fatal("expected connect error")
	}
}

func TestFixMonitoringTablesCollationSkipsSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(testutil.MemoryDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	fixMonitoringTablesCollation(db) // must not error on non-MySQL
}

func TestEnsureSchemaAdditionsSkipsMissingTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(testutil.MemoryDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ensureSchemaAdditions(db) // empty DB — must not panic
}

func TestEnsureSchemaAdditionsAddsSQLiteColumns(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	ensureSchemaAdditions(db)
	m := db.Migrator()
	if !m.HasColumn(&models.GameServer{}, "content_resources") && !columnExists(db, "game_servers", "content_resources") {
		t.Fatal("expected content_resources on game_servers")
	}
	if !columnExists(db, "prepare_requests", "progress_message") {
		t.Fatal("expected progress_message on prepare_requests")
	}
	if !columnExists(db, "launch_requests", "progress_message") {
		t.Fatal("expected progress_message on launch_requests")
	}
	if !columnExists(db, "launcher_instances", "managed_by_game_server_id") {
		t.Fatal("expected managed_by_game_server_id on launcher_instances")
	}
}

func TestDropResourceBlobColumnsSkipsSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(testutil.MemoryDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	dropResourceBlobColumns(db) // must not error on non-MySQL
}

func TestDropRedundantUsersEmailIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(testutil.MemoryDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		tier TEXT NOT NULL DEFAULT 'free',
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_users_email ON users (email)`).Error; err != nil {
		t.Fatalf("create index: %v", err)
	}

	dropRedundantUsersEmailIndex(db)

	m := db.Migrator()
	if m.HasIndex(&models.User{}, "idx_users_email") {
		t.Fatal("expected legacy idx_users_email to be dropped")
	}
}

func TestOpenSQLiteAndPing(t *testing.T) {
	db, err := Open(sqlite.Open(testutil.MemoryDSN(t)))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Ping(db); err != nil {
		t.Fatalf("ping: %v", err)
	}
	// Trigger GORM NowFunc configured in Open.
	if err := db.Create(&models.User{
		ID:           "now-func-user-" + t.Name(),
		Email:        "now-" + t.Name() + "@test.com",
		PasswordHash: "hash",
		Tier:         "free",
	}).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
}

func TestOpenMigrateFailure(t *testing.T) {
	old := migrateUsers
	migrateUsers = func(*gorm.DB) error {
		return errors.New("migrate failed")
	}
	t.Cleanup(func() { migrateUsers = old })

	if _, err := Open(sqlite.Open("file:migratefail?mode=memory&cache=shared")); err == nil {
		t.Fatal("expected migrate error")
	}
}

func TestConfigurePoolDBHandleError(t *testing.T) {
	old := poolDB
	poolDB = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("db handle error")
	}
	t.Cleanup(func() { poolDB = old })

	if _, err := Open(sqlite.Open("file:pooldbfail?mode=memory&cache=shared")); err == nil {
		t.Fatal("expected configure pool error")
	}
}

func TestConfigurePoolError(t *testing.T) {
	old := configurePool
	configurePool = func(*gorm.DB) error {
		return errors.New("pool failed")
	}
	t.Cleanup(func() { configurePool = old })

	if _, err := Open(sqlite.Open("file:poolfail?mode=memory&cache=shared")); err == nil {
		t.Fatal("expected pool error")
	}
}

func TestPingErrors(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	testutil.CloseDB(t, db)
	if err := Ping(db); err == nil {
		t.Fatal("expected ping error on closed db")
	}
}

func TestPingDBError(t *testing.T) {
	old := poolDB
	poolDB = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("db handle error")
	}
	t.Cleanup(func() { poolDB = old })

	db := testutil.OpenSQLiteDB(t)
	if err := Ping(db); err == nil {
		t.Fatal("expected ping db error")
	}
}
