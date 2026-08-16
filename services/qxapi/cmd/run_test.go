package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/config"
	"github.com/qxproject/qx/services/qxapi/internal/database"
)

func chdirRepo(t *testing.T, toml string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxapi")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if toml != "" {
		if err := os.WriteFile(filepath.Join(root, "qxapi.toml"), []byte(toml), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestRunInvalidDatabase(t *testing.T) {
	chdirRepo(t, "database_dsn = \"invalid-dsn\"\n")
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
	cfg.MinioEndpoint = "memory"
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

	chdirRepo(t, "addr = \"invalid-address\"\nminio_endpoint = \"memory\"\n")
	if err := run(); err == nil {
		t.Fatal("expected listen error")
	}
}
