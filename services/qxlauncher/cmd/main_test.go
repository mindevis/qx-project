package main

import (
	"os"
	"path/filepath"
	"testing"
)

func setupLauncherTestRepo(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxlauncher")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(t.TempDir(), "device_token")
	toml := `skip_tray = true
link_max_polls = 0
api_base_url = "http://127.0.0.1:1/api/v1"
device_token_path = "` + filepath.ToSlash(tokenPath) + `"
`
	if err := os.WriteFile(filepath.Join(root, "launcher.toml"), []byte(toml), 0o600); err != nil {
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
}

func TestMainRunsWithoutTray(t *testing.T) {
	setupLauncherTestRepo(t)
	main()
}
