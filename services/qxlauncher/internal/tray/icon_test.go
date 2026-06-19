package tray

import "testing"

func TestLauncherPageURL(t *testing.T) {
	if got := LauncherPageURL("http://localhost:5173/"); got != "http://localhost:5173/launcher" {
		t.Fatalf("got %q", got)
	}
	if got := LauncherPageURL(""); got != "http://localhost:5173/launcher" {
		t.Fatalf("default: %q", got)
	}
}
