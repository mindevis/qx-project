package minecraft

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstanceFSListReadWrite(t *testing.T) {
	root := t.TempDir()
	dl := NewDownloader(root)
	instanceID := "test-inst"
	gameDir, err := dl.InstanceGameDir(instanceID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(gameDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(gameDir, "config", "test.toml")
	if err := os.WriteFile(configPath, []byte("key=value"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := dl.ListInstanceDir(instanceID, "config")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "test.toml" {
		t.Fatalf("entries: %+v", entries)
	}

	content, size, err := dl.ReadInstanceFile(instanceID, "config/test.toml")
	if err != nil || content != "key=value" || size == 0 {
		t.Fatalf("read: content=%q size=%d err=%v", content, size, err)
	}

	if err := dl.WriteInstanceFile(instanceID, "config/test.toml", "updated=1"); err != nil {
		t.Fatalf("write: %v", err)
	}
	content, _, err = dl.ReadInstanceFile(instanceID, "config/test.toml")
	if err != nil || content != "updated=1" {
		t.Fatalf("after write: %q err=%v", content, err)
	}
}

func TestCloneInstanceDir(t *testing.T) {
	root := t.TempDir()
	dl := NewDownloader(root)
	srcID := "src-inst"
	destID := "dest-inst"
	srcDir, err := dl.InstanceRoot(srcID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "mods", "a.jar"), []byte("mod"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "session.lock"), []byte("lock"), 0o644); err != nil {
		t.Fatal(err)
	}

	copied, err := dl.CloneInstanceDir(srcID, destID)
	if err != nil || !copied {
		t.Fatalf("clone: copied=%v err=%v", copied, err)
	}
	destDir, err := dl.InstanceRoot(destID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "mods", "a.jar")); err != nil {
		t.Fatalf("copied jar missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destDir, "session.lock")); !os.IsNotExist(err) {
		t.Fatal("session.lock should not be copied")
	}

	copied, err = dl.CloneInstanceDir("missing-inst", "dest-missing")
	if err != nil || copied {
		t.Fatalf("missing src: copied=%v err=%v", copied, err)
	}
	if _, err := dl.CloneInstanceDir(srcID, srcID); err == nil {
		t.Fatal("expected same-id error")
	}
}

func TestReadInstanceResourceFile(t *testing.T) {
	root := t.TempDir()
	dl := NewDownloader(root)
	instanceID := "test-inst"
	if err := dl.WriteInstanceResourceFile(instanceID, "mods", "example.jar", []byte("mod-bytes")); err != nil {
		t.Fatalf("write resource: %v", err)
	}
	data, err := dl.ReadInstanceResourceFile(instanceID, "mods", "example.jar")
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if string(data) != "mod-bytes" {
		t.Fatalf("data=%q", data)
	}
}
