package tray

import (
	"bytes"
	"image/png"
	"runtime"
	"testing"
)

func TestLauncherPageURL(t *testing.T) {
	if got := LauncherPageURL("http://localhost:5173/"); got != "http://localhost:5173/launcher" {
		t.Fatalf("got %q", got)
	}
	if got := LauncherPageURL(""); got != "http://localhost:5173/launcher" {
		t.Fatalf("default: %q", got)
	}
}

func TestBrandTrayIcon(t *testing.T) {
	if len(trayIconData) < 8 {
		t.Fatal("empty tray icon")
	}
	if runtime.GOOS == "windows" {
		if trayIconData[0] != 0 || trayIconData[1] != 0 || trayIconData[2] != 1 || trayIconData[3] != 0 {
			t.Fatalf("windows tray icon must be ICO, got %x", trayIconData[:4])
		}
		return
	}
	if string(trayIconData[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("tray icon is not a PNG")
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(trayIconData))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Width != 32 || cfg.Height != 32 {
		t.Fatalf("expected 32x32 icon, got %dx%d", cfg.Width, cfg.Height)
	}
}

func TestTrayStateIcons(t *testing.T) {
	if len(iconPendingPNG) == 0 || len(iconLinkedPNG) == 0 {
		t.Fatal("tray state icons must be set")
	}
}
