package cache

import (
	"testing"

	"os"
	"path/filepath"

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

func TestPruneInstanceDataRemovesDeletedInstances(t *testing.T) {
	dir := t.TempDir()
	removed := filepath.Join(InstanceDataRoot(dir), "inst-old")
	kept := filepath.Join(InstanceDataRoot(dir), "inst-keep")
	if err := os.MkdirAll(removed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(removed, "marker.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(kept, 0o755); err != nil {
		t.Fatal(err)
	}

	items := []apiclient.InstanceItem{
		{ID: "inst-keep", Name: "Keep", MCVersion: "1.21", Loader: "vanilla"},
	}
	if err := PruneInstanceData(dir, items); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if _, err := os.Stat(removed); !os.IsNotExist(err) {
		t.Fatalf("removed dir still exists: %v", err)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("kept dir missing: %v", err)
	}
}

func TestSyncInstancesPrunesAndSaves(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(InstanceDataRoot(dir), "inst-stale")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	items := []apiclient.InstanceItem{
		{ID: "inst-1", Name: "Survival", MCVersion: "1.21", Loader: "vanilla"},
	}
	if err := SyncInstances(dir, items); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale dir still exists: %v", err)
	}
	snap, err := LoadInstances(dir)
	if err != nil || len(snap.Instances) != 1 || snap.Instances[0].ID != "inst-1" {
		t.Fatalf("snapshot: err=%v snap=%+v", err, snap)
	}
}
