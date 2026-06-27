package minecraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGameLanguageCreatesOptions(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureGameLanguage(dir, "ru_ru"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "options.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "lang:ru_ru") {
		t.Fatalf("options: %q", data)
	}
}

func TestEnsureGameLanguageUpdatesExistingLang(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "options.txt")
	if err := os.WriteFile(path, []byte("lang:en_us\nsoundCategory_master:0.5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureGameLanguage(dir, "ru_ru"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	text := string(data)
	if strings.Contains(text, "lang:en_us") {
		t.Fatalf("lang not updated: %q", text)
	}
	if !strings.Contains(text, "lang:ru_ru") || !strings.Contains(text, "soundCategory_master:0.5") {
		t.Fatalf("options: %q", text)
	}
}

func TestEnsureGameLanguageDefault(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureGameLanguage(dir, ""); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "options.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "lang:ru_ru") {
		t.Fatalf("options: %q", data)
	}
}
