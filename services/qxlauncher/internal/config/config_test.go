package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromRepoTOML(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxlauncher")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "launcher.toml"), []byte(`api_base_url = "http://example.test/api/v1"
web_base_url = "http://example.test"
`), 0o600); err != nil {
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

	cfg := Load()
	if cfg.APIBaseURL != "http://example.test/api/v1" {
		t.Fatalf("api base: got %q", cfg.APIBaseURL)
	}
	if cfg.WebBaseURL != "http://example.test" {
		t.Fatalf("web base: got %q", cfg.WebBaseURL)
	}
}

func TestLoadFromUserTOML(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	qxDir := filepath.Join(home, DefaultDataDirName)
	if err := os.MkdirAll(qxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	toml := `api_base_url = "http://toml.test/api/v1"
skip_tray = true
`
	if err := os.WriteFile(filepath.Join(qxDir, "launcher.toml"), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	cfg := Load()
	if cfg.APIBaseURL != "http://toml.test/api/v1" {
		t.Fatalf("got %q", cfg.APIBaseURL)
	}
	if !cfg.SkipTray {
		t.Fatal("expected skip_tray from toml")
	}
}

func TestDefaultsWithoutEmbedded(t *testing.T) {
	if embeddedAPIBaseURL != "" || embeddedWebBaseURL != "" {
		t.Skip("test binary built with prod ldflags")
	}
	cfg := defaults()
	if cfg.APIBaseURL != defaultDevAPIBaseURL {
		t.Fatalf("api base: got %q want %q", cfg.APIBaseURL, defaultDevAPIBaseURL)
	}
	if cfg.WebBaseURL != defaultDevWebBaseURL {
		t.Fatalf("web base: got %q want %q", cfg.WebBaseURL, defaultDevWebBaseURL)
	}
}

func TestEmbeddedProdDefaults(t *testing.T) {
	if embeddedAPIBaseURL == "" || embeddedWebBaseURL == "" {
		t.Skip("re-run with LAUNCHER_PROD_LDFLAGS (see Makefile build-launcher-win)")
	}
	cfg := defaults()
	if cfg.APIBaseURL != embeddedAPIBaseURL {
		t.Fatalf("api base: got %q want %q", cfg.APIBaseURL, embeddedAPIBaseURL)
	}
	if cfg.WebBaseURL != embeddedWebBaseURL {
		t.Fatalf("web base: got %q want %q", cfg.WebBaseURL, embeddedWebBaseURL)
	}
}
