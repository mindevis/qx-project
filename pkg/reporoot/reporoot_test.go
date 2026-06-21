package reporoot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAndConfigFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxapi")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "qxapi.toml"), []byte("addr=\":3000\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	found, err := Find(".")
	if err != nil {
		t.Fatal(err)
	}
	if found != root {
		t.Fatalf("root: got %q want %q", found, root)
	}
	if ConfigFile(".", "qxapi.toml") != filepath.Join(root, "qxapi.toml") {
		t.Fatal("expected qxapi.toml path")
	}
	if ConfigFile(".", "missing.toml") != "" {
		t.Fatal("expected empty for missing file")
	}
}
