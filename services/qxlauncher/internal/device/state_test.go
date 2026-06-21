package device

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveDeviceID(t *testing.T) {
	dir := t.TempDir()
	if got := LoadDeviceID(dir); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if err := SaveDeviceID(dir, "dev-1"); err != nil {
		t.Fatal(err)
	}
	if got := LoadDeviceID(dir); got != "dev-1" {
		t.Fatalf("device id: %q", got)
	}
}

func TestLoadDeviceIDFromEnv(t *testing.T) {
	t.Setenv("QX_DEVICE_ID", "env-device")
	if got := LoadDeviceID(t.TempDir()); got != "env-device" {
		t.Fatalf("env device id: %q", got)
	}
}

func TestReadTokenTrimsWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("jwt-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ReadToken(path); got != "jwt-token" {
		t.Fatalf("token: %q", got)
	}
}
