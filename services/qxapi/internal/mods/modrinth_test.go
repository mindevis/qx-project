package mods

import "testing"

func TestModrinthSearchIconURL(t *testing.T) {
	t.Parallel()
	if got := modrinthSearchIconURL("https://cdn/icon.png", ""); got != "https://cdn/icon.png" {
		t.Fatalf("icon_url: got %q", got)
	}
	if got := modrinthSearchIconURL("", "https://cdn/legacy.png"); got != "https://cdn/legacy.png" {
		t.Fatalf("display_icon_url fallback: got %q", got)
	}
	if got := modrinthSearchIconURL("https://cdn/new.png", "https://cdn/old.png"); got != "https://cdn/new.png" {
		t.Fatalf("prefer icon_url: got %q", got)
	}
}
