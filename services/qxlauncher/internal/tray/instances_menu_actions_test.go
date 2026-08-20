package tray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/cache"
)

func TestInstancesMenuHandleDeleteUpdatesCache(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/instances/inst-del":
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	items := []apiclient.InstanceItem{
		{ID: "inst-del", Name: "Remove me", MCVersion: "1.21", Loader: "vanilla"},
		{ID: "inst-keep", Name: "Keep", MCVersion: "1.21", Loader: "vanilla"},
	}
	if err := os.MkdirAll(filepath.Join(dir, "instances", "inst-del"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cache.SaveInstances(dir, items); err != nil {
		t.Fatal(err)
	}

	menu := NewInstancesMenu(InstancesMenuConfig{
		APIBase:   srv.URL,
		UserToken: "user-jwt",
		DataDir:   dir,
	})
	menu.handleDelete(items[0])

	if !deleteCalled {
		t.Fatal("expected delete API call")
	}
	snap, err := cache.LoadInstances(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Instances) != 1 || snap.Instances[0].ID != "inst-keep" {
		t.Fatalf("cache after delete: %+v", snap.Instances)
	}
	if _, err := os.Stat(filepath.Join(dir, "instances", "inst-del")); !os.IsNotExist(err) {
		t.Fatalf("pruned dir still exists: %v", err)
	}
}

func TestInstancesMenuHandleDeleteRemovesFilesWithoutCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/instances/inst-del" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	removed := filepath.Join(dir, "instances", "inst-del")
	if err := os.MkdirAll(removed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(removed, "world.dat"), []byte("save"), 0o644); err != nil {
		t.Fatal(err)
	}

	menu := NewInstancesMenu(InstancesMenuConfig{
		APIBase:   srv.URL,
		UserToken: "user-jwt",
		DataDir:   dir,
	})
	menu.handleDelete(apiclient.InstanceItem{ID: "inst-del", Name: "Remove me"})
	if _, err := os.Stat(removed); !os.IsNotExist(err) {
		t.Fatalf("instance files still exist: %v", err)
	}
}

func TestInstancesMenuHandleLaunchRequiresUserToken(t *testing.T) {
	menu := NewInstancesMenu(InstancesMenuConfig{
		APIBase: "http://127.0.0.1:1",
		DataDir: t.TempDir(),
	})
	menu.handleLaunch(apiclient.InstanceItem{ID: "inst-1", Name: "Test"})
}

func TestSyncInstancesCallback(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/launcher/devices/me/instances":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "1", "name": "A", "mc_version": "1.21", "loader": "vanilla"},
				},
			})
		case "/launcher/update-requests/pending":
			_ = json.NewEncoder(w).Encode(map[string]any{"item": nil})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiSrv.Close)

	dir := t.TempDir()
	var synced []apiclient.InstanceItem
	cfg := Config{
		APIBase:           apiSrv.URL,
		DeviceToken:       "token",
		DataDir:           dir,
		LaunchPoll:        1 * time.Hour,
		InstancePoll:      1 * time.Millisecond,
		OnInstancesSynced: func(items []apiclient.InstanceItem) { synced = items },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go RunLoop(ctx, cfg)

	deadline := time.Now().Add(400 * time.Millisecond)
	for len(synced) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)
	if len(synced) != 1 || synced[0].Name != "A" {
		t.Fatalf("sync callback: %+v", synced)
	}
}
