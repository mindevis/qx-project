package tray

import (
	"testing"

	"github.com/qxproject/qx/pkg/safepath"
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
	root := t.TempDir()
	got := instanceDir(root, "inst-1")
	want, err := safepath.Join(root, "instances", "inst-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("dir = %q, want %q", got, want)
	}
	if instanceDir(root, "..") != "" {
		t.Fatal("expected empty path for traversal id")
	}
}
