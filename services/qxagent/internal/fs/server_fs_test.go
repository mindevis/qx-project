package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWipeWorkDirRemovesAllContent(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "instance-1")
	if err := os.MkdirAll(filepath.Join(workDir, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "server.properties"), []byte("motd=test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "mods", "example.jar"), []byte("mod"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WipeWorkDir(workDir); err != nil {
		t.Fatalf("wipe: %v", err)
	}

	info, err := os.Stat(workDir)
	if err != nil {
		t.Fatalf("work dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("work dir is not a directory")
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty work dir, got %d entries", len(entries))
	}
}

func TestWipeWorkDirMissingDirCreatesEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "new-instance")

	if err := WipeWorkDir(workDir); err != nil {
		t.Fatalf("wipe missing dir: %v", err)
	}
	info, err := os.Stat(workDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected created dir: err=%v", err)
	}
}

func TestListClientModsMissingDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ListClientMods(dir, "forge")
	if err != nil {
		t.Fatalf("list client mods: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(entries))
	}
}
