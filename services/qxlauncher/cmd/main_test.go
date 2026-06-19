package main

import (
	"path/filepath"
	"testing"
)

func TestMainRunsWithoutTray(t *testing.T) {
	t.Setenv("QX_LINK_MAX_POLLS", "0")
	t.Setenv("QX_SKIP_TRAY", "1")
	t.Setenv("QX_DEVICE_TOKEN_PATH", filepath.Join(t.TempDir(), "device_token"))
	t.Setenv("QX_API_BASE_URL", "http://127.0.0.1:1/api/v1")
	main()
}
