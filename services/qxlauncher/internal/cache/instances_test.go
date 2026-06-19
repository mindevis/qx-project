package cache

import (
	"testing"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
)

func TestSaveLoadInstances(t *testing.T) {
	dir := t.TempDir()
	items := []apiclient.InstanceItem{
		{ID: "1", Name: "Survival", MCVersion: "1.21", Loader: "vanilla"},
	}
	if err := SaveInstances(dir, items); err != nil {
		t.Fatalf("save: %v", err)
	}
	snap, err := LoadInstances(dir)
	if err != nil || len(snap.Instances) != 1 {
		t.Fatalf("load: err=%v snap=%+v", err, snap)
	}
}

func TestSaveInstancesEmptySlice(t *testing.T) {
	dir := t.TempDir()
	if err := SaveInstances(dir, nil); err != nil {
		t.Fatalf("save nil: %v", err)
	}
	snap, err := LoadInstances(dir)
	if err != nil || len(snap.Instances) != 0 {
		t.Fatalf("load empty: err=%v snap=%+v", err, snap)
	}
}
