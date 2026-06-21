package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSetsUnsetKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("FOO=bar\n# comment\nBAZ=\"qux\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FOO", "existing")
	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FOO") != "existing" {
		t.Fatalf("should not override existing env")
	}
	if os.Getenv("BAZ") != "qux" {
		t.Fatalf("got BAZ=%q", os.Getenv("BAZ"))
	}
}

func TestLoadRepoDotEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.22\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxapi")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("FROM_ENV=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadRepoDotEnv(sub); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FROM_ENV") != "1" {
		t.Fatal("expected FROM_ENV from repo .env")
	}
}
