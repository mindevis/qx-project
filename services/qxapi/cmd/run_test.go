package main

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/config"
	"github.com/qxproject/qx/services/qxapi/internal/database"
)

func TestRunInvalidDatabase(t *testing.T) {
	t.Setenv("DATABASE_DSN", "invalid-dsn")
	if err := run(); err == nil {
		t.Fatal("expected database error")
	}
}

func TestBootstrapWithSQLite(t *testing.T) {
	old := connectDB
	connectDB = func(_ string) (*gorm.DB, error) {
		return database.Open(sqlite.Open("file:bootstrap?mode=memory&cache=shared"))
	}
	t.Cleanup(func() { connectDB = old })

	cfg := config.Load()
	router, err := bootstrap(cfg)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if router == nil {
		t.Fatal("expected router")
	}
}

func TestRunListenError(t *testing.T) {
	old := connectDB
	connectDB = func(_ string) (*gorm.DB, error) {
		return database.Open(sqlite.Open("file:runfail?mode=memory&cache=shared"))
	}
	t.Cleanup(func() { connectDB = old })

	t.Setenv("API_ADDR", "invalid-address")
	if err := run(); err == nil {
		t.Fatal("expected listen error")
	}
}
