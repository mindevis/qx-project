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
	modPath, err := ContentRelPath(dir, "forge", "mod", "example.jar", "")
	if err != nil {
		t.Fatal(err)
	}
	if modPath != "mods/example.jar" {
		t.Fatalf("mod path: %q", modPath)
	}
	clientModPath, err := ContentRelPath(dir, "forge", "mod", "example.jar", "client-mods")
	if err != nil {
		t.Fatal(err)
	}
	if clientModPath != "client-mods/example.jar" {
		t.Fatalf("client mod path: %q", clientModPath)
	}
	resourcepackPath, err := ContentRelPath(dir, "forge", "resourcepack", "pack.zip", "")
	if err != nil {
		t.Fatal(err)
	}
	if resourcepackPath != "resourcepacks/pack.zip" {
		t.Fatalf("resourcepack path: %q", resourcepackPath)
	}
	clientResourcepackPath, err := ContentRelPath(dir, "forge", "resourcepack", "pack.zip", "client-resourcepacks")
	if err != nil {
		t.Fatal(err)
	}
	if clientResourcepackPath != "client-resourcepacks/pack.zip" {
		t.Fatalf("client resourcepack path: %q", clientResourcepackPath)
	}
	shaderPath, err := ContentRelPath(dir, "forge", "shader", "pack.zip", "")
	if err != nil {
		t.Fatal(err)
	}
	if shaderPath != "shaderpacks/pack.zip" {
		t.Fatalf("shader path: %q", shaderPath)
	}
	clientShaderPath, err := ContentRelPath(dir, "forge", "shader", "pack.zip", "client-shaders")
	if err != nil {
		t.Fatal(err)
	}
	if clientShaderPath != "client-shaders/pack.zip" {
		t.Fatalf("client shader path: %q", clientShaderPath)
	}
	pluginPath, err := ContentRelPath(dir, "paper", "plugin", "EssentialsX.jar", "")
	if err != nil {
		t.Fatal(err)
	}
	if pluginPath != "plugins/EssentialsX.jar" {
		t.Fatalf("plugin path: %q", pluginPath)
	}
	datapackPath, err := ContentRelPath(dir, "vanilla", "datapack", "pack.zip", "")
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

func TestSanitizeContentDownloadURL(t *testing.T) {
	if _, err := sanitizeContentDownloadURL("http://127.0.0.1/evil.jar"); err == nil {
		t.Fatal("expected localhost rejected")
	}
	if _, err := sanitizeContentDownloadURL("https://evil.example/mod.jar"); err == nil {
		t.Fatal("expected unknown host rejected")
	}
	if _, err := sanitizeContentDownloadURL("file:///etc/passwd"); err == nil {
		t.Fatal("expected file url rejected")
	}
	got, err := sanitizeContentDownloadURL("https://cdn.modrinth.com/data/AANobbMI/versions/foo/sodium.jar")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://cdn.modrinth.com/data/AANobbMI/versions/foo/sodium.jar" {
		t.Fatalf("url: %s", got)
	}
	wantTAB := "https://cdn.modrinth.com/data/gG7VFbG0/versions/Za7G9fdJ/TAB%20v6.1.2.jar"
	for _, raw := range []string{
		wantTAB,
		"https://cdn.modrinth.com/data/gG7VFbG0/versions/Za7G9fdJ/TAB%2520v6.1.2.jar",
	} {
		spaced, err := sanitizeContentDownloadURL(raw)
		if err != nil {
			t.Fatal(err)
		}
		if spaced != wantTAB {
			t.Fatalf("spaced url from %s: %s", raw, spaced)
		}
	}
	hangarURL, err := sanitizeContentDownloadURL("https://hangarcdn.papermc.io/plugins/dmulloy2/ProtocolLib/versions/5.1.0/PAPER/ProtocolLib.jar")
	if err != nil {
		t.Fatal(err)
	}
	if hangarURL == "" {
		t.Fatal("expected hangar cdn url")
	}
	spigetURL, err := sanitizeContentDownloadURL("https://api.spiget.org/v2/resources/28140/versions/123/download")
	if err != nil {
		t.Fatal(err)
	}
	if spigetURL == "" {
		t.Fatal("expected spiget url")
	}
	cursecdnURL, err := sanitizeContentDownloadURL("https://media-elerium.cursecdn.com/files/vault.jar")
	if err != nil {
		t.Fatal(err)
	}
	if cursecdnURL == "" {
		t.Fatal("expected cursecdn url")
	}
}

func TestReadContentFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "client-mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "client-mods", "journeymap.jar")
	if err := os.WriteFile(path, []byte("fake-jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := ReadContentFile(dir, "forge", "mod", "client-mods", "journeymap.jar")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-jar" {
		t.Fatalf("content: %q", string(data))
	}
}
