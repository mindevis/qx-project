package testutil

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrateUsersDefault(t *testing.T) {
	db := OpenSQLiteDB(t)
	if err := autoMigrateUsers(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestCloseSQLSuccess(t *testing.T) {
	db := OpenSQLiteDB(t)
	if err := closeSQL(db); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestOpenSQLiteDBDirect(t *testing.T) {
	db, err := openSQLiteDB("direct-success")
	if err != nil || db == nil {
		t.Fatalf("openSQLiteDB: err=%v db=%v", err, db)
	}
}

func TestOpenSQLiteDBAndClose(t *testing.T) {
	db := OpenSQLiteDB(t)
	if db == nil {
		t.Fatal("expected db")
	}
	CloseDB(t, db)
	CloseDB(t, db)
	CloseDB(t, nil)

	old := closeSQL
	closeSQL = func(db *gorm.DB) error {
		_ = db
		return errors.New("close failed")
	}
	t.Cleanup(func() { closeSQL = old })
	CloseDB(t, OpenSQLiteDB(t))
}

type fatalRecorder struct {
	testing.TB
	fatal bool
}

func (f *fatalRecorder) Fatalf(format string, args ...any) {
	f.fatal = true
}

func TestOpenSQLiteDBFatalf(t *testing.T) {
	old := openGORM
	t.Cleanup(func() { openGORM = old })
	openGORM = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}

	rec := &fatalRecorder{TB: t}
	OpenSQLiteDB(rec)
	if !rec.fatal {
		t.Fatal("expected OpenSQLiteDB to fail")
	}
}

func TestOpenSQLiteDBErrors(t *testing.T) {
	oldGORM := openGORM
	t.Cleanup(func() { openGORM = oldGORM })
	openGORM = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
		return nil, errors.New("open failed")
	}
	if _, err := openSQLiteDB("bad-open"); err == nil {
		t.Fatal("expected open error")
	}
	openGORM = oldGORM

	oldMigrate := autoMigrateUsers
	t.Cleanup(func() { autoMigrateUsers = oldMigrate })
	autoMigrateUsers = func(*gorm.DB) error {
		return errors.New("migrate failed")
	}
	if _, err := openSQLiteDB("bad-migrate"); err == nil {
		t.Fatal("expected migrate error")
	}
}

func TestCloseSQLDBHandleError(t *testing.T) {
	old := gormSQLDB
	gormSQLDB = func(*gorm.DB) (*sql.DB, error) {
		return nil, errors.New("db handle error")
	}
	t.Cleanup(func() { gormSQLDB = old })

	if err := closeSQL(OpenSQLiteDB(t)); err == nil {
		t.Fatal("expected db handle error")
	}
}

func TestAutoMigrateUsersError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migratehook?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	old := autoMigrateUsers
	autoMigrateUsers = func(*gorm.DB) error {
		return errors.New("migrate failed")
	}
	t.Cleanup(func() { autoMigrateUsers = old })
	if err := autoMigrateUsers(db); err == nil {
		t.Fatal("expected migrate error")
	}
}
