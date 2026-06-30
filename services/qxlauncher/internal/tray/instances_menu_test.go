package tray

import (
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
)

func TestInstanceMenuTitle(t *testing.T) {
	title := instanceMenuTitle(apiclient.InstanceItem{
		ID:        "id-1",
		Name:      "Survival",
		MCVersion: "1.21",
	})
	if title != "Survival (1.21)" {
		t.Fatalf("unexpected title: %q", title)
	}
	title = instanceMenuTitle(apiclient.InstanceItem{ID: "id-2"})
	if title != "id-2" {
		t.Fatalf("fallback title: %q", title)
	}
}

func TestInstanceDir(t *testing.T) {
	got := instanceDir("/data", "inst-1")
	want := filepath.Join("/data", "instances", "inst-1")
	if got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
}
