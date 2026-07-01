package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorldFolderDefault(t *testing.T) {
	dir := t.TempDir()
	world, err := WorldFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if world != "world" {
		t.Fatalf("expected world, got %q", world)
	}
}

func TestWorldFolderFromProperties(t *testing.T) {
	dir := t.TempDir()
	props := filepath.Join(dir, "server.properties")
	if err := os.WriteFile(props, []byte("level-name=custom_world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	world, err := WorldFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if world != "custom_world" {
		t.Fatalf("expected custom_world, got %q", world)
	}
}

func TestContentRelPath(t *testing.T) {
	dir := t.TempDir()
	modPath, err := ContentRelPath(dir, "forge", "mod", "example.jar")
	if err != nil {
		t.Fatal(err)
	}
	if modPath != "mods/example.jar" {
		t.Fatalf("mod path: %q", modPath)
	}
	pluginPath, err := ContentRelPath(dir, "paper", "plugin", "EssentialsX.jar")
	if err != nil {
		t.Fatal(err)
	}
	if pluginPath != "plugins/EssentialsX.jar" {
		t.Fatalf("plugin path: %q", pluginPath)
	}
	datapackPath, err := ContentRelPath(dir, "vanilla", "datapack", "pack.zip")
	if err != nil {
		t.Fatal(err)
	}
	if datapackPath != "world/datapacks/pack.zip" {
		t.Fatalf("datapack path: %q", datapackPath)
	}
}

func TestListPluginsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := ListPlugins(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty, got %d", len(entries))
	}
}

func TestListDatapacksEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "world", "datapacks"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := ListDatapacks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty, got %d", len(entries))
	}
}
