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
