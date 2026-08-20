package tray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/minecraft"
)

type stubManifestBuilder struct {
	manifest *mcmanifest.InstanceLaunchManifest
}

func (s stubManifestBuilder) BuildInstanceManifest(_ context.Context, instanceID, name, mcVersion, loader, loaderVersion, targetOS string) (*mcmanifest.InstanceLaunchManifest, error) {
	if s.manifest != nil {
		m := *s.manifest
		if m.InstanceID == "" {
			m.InstanceID = instanceID
		}
		return &m, nil
	}
	return &mcmanifest.InstanceLaunchManifest{
		InstanceID: instanceID,
		MCVersion:  mcVersion,
		MainClass:  "Main",
	}, nil
}

func TestExecuteLaunchDryRun(t *testing.T) {
	jarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("jar-bytes"))
	}))
	t.Cleanup(jarSrv.Close)

	indexBody, _ := json.Marshal(map[string]any{
		"objects": map[string]any{},
	})
	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(indexBody)
	}))
	t.Cleanup(indexSrv.Close)

	var lastStatus string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["status"].(string); ok {
			lastStatus = s
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "req-1", "status": lastStatus})
	}))
	t.Cleanup(apiSrv.Close)

	api := apiclient.New(apiSrv.URL, "token")
	dl := minecraft.NewDownloader(filepath.Join(t.TempDir(), "data"))
	dl.SkipJavaDownload = true
	dl.ManifestBuilder = stubManifestBuilder{manifest: &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		MCVersion:  "1.21",
		MainClass:  "Main",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.21", URL: indexSrv.URL},
		ClientJar:  mcmanifest.DownloadFile{URL: jarSrv.URL, Sha1: ""},
	}}

	executeLaunch(context.Background(), api, dl, Config{LaunchDryRun: true}, &apiclient.LaunchRequestItem{
		ID: "req-1",
		Profile: &apiclient.OfflineProfile{
			Username:    "Steve",
			OfflineUUID: "00000000-0000-0000-0000-000000000001",
		},
		Instance: &apiclient.LaunchInstance{
			ID:        "inst-1",
			Name:      "Survival",
			MCVersion: "1.21",
			Loader:    mcmanifest.LoaderVanilla,
		},
	})

	if lastStatus != "completed" {
		t.Fatalf("expected completed, got %q", lastStatus)
	}
}

func TestExecuteLaunchDryRunNeoForge(t *testing.T) {
	jarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("jar-bytes"))
	}))
	t.Cleanup(jarSrv.Close)

	indexBody, _ := json.Marshal(map[string]any{
		"objects": map[string]any{},
	})
	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(indexBody)
	}))
	t.Cleanup(indexSrv.Close)

	var lastStatus string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["status"].(string); ok {
			lastStatus = s
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "req-neoforge", "status": lastStatus})
	}))
	t.Cleanup(apiSrv.Close)

	dataDir := filepath.Join(t.TempDir(), "data")
	instanceID := "inst-neoforge"
	clientJar := mcmanifest.DefaultLoaderClientJar(mcmanifest.LoaderNeoForge, "1.21.1", "21.1.234")
	instanceRoot := filepath.Join(dataDir, "instances", instanceID)
	if err := os.MkdirAll(filepath.Join(instanceRoot, filepath.Dir(clientJar.RelativePath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instanceRoot, clientJar.RelativePath), []byte("neoforge-client"), 0o644); err != nil {
		t.Fatal(err)
	}

	api := apiclient.New(apiSrv.URL, "token")
	dl := minecraft.NewDownloader(dataDir)
	dl.SkipJavaDownload = true
	dl.ManifestBuilder = stubManifestBuilder{manifest: &mcmanifest.InstanceLaunchManifest{
		InstanceID:    "inst-neoforge",
		MCVersion:     "1.21.1",
		Loader:        mcmanifest.LoaderNeoForge,
		LoaderVersion: "21.1.234",
		VersionID:     "1.21.1-neoforge-21.1.234",
		MainClass:     "cpw.mods.bootstraplauncher.BootstrapLauncher",
		AssetIndex:    mcmanifest.AssetIndexRef{ID: "1.21.1", URL: indexSrv.URL},
		ClientJar:     mcmanifest.DownloadFile{URL: jarSrv.URL, Sha1: ""},
		LoaderClientJar: clientJar,
		JVMArguments: []string{
			"-DlibraryDirectory=${library_directory}",
			"-p", "${library_directory}/cpw/mods/bootstraplauncher/2.0.2/bootstraplauncher-2.0.2.jar",
		},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--launchTarget", "forgeclient",
		},
	}}

	executeLaunch(context.Background(), api, dl, Config{LaunchDryRun: true}, &apiclient.LaunchRequestItem{
		ID: "req-neoforge",
		Profile: &apiclient.OfflineProfile{
			Username:    "NeoForgeTest",
			OfflineUUID: "00000000-0000-0000-0000-000000000001",
		},
		Instance: &apiclient.LaunchInstance{
			ID:            "inst-neoforge",
			Name:          "NeoForge",
			MCVersion:     "1.21.1",
			Loader:        mcmanifest.LoaderNeoForge,
			LoaderVersion: "21.1.234",
		},
	})

	if lastStatus != "completed" {
		t.Fatalf("expected completed, got %q", lastStatus)
	}
}

func TestSyncInstances(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/devices/me/instances" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "1", "name": "Survival", "mc_version": "1.21", "loader": "vanilla"},
			},
		})
	}))
	t.Cleanup(apiSrv.Close)

	dir := t.TempDir()
	stale := filepath.Join(dir, "instances", "stale-id")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	api := apiclient.New(apiSrv.URL, "token")
	items, err := syncInstances(context.Background(), api, dir)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Survival" {
		t.Fatalf("sync items: %+v", items)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale instance dir still exists: %v", err)
	}
}

func TestRunLoopPrunesDeletedInstancesOnStart(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/launcher/devices/me/instances":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{
					{"id": "inst-keep", "name": "Keep", "mc_version": "1.21", "loader": "vanilla"},
				},
			})
		case "/launcher/launch-requests/pending", "/launcher/update-requests/pending":
			_ = json.NewEncoder(w).Encode(map[string]any{"item": nil})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiSrv.Close)

	dir := t.TempDir()
	stale := filepath.Join(dir, "instances", "inst-stale")
	kept := filepath.Join(dir, "instances", "inst-keep")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(kept, 0o755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go RunLoop(ctx, Config{
		APIBase:      apiSrv.URL,
		DeviceToken:  "token",
		DataDir:      dir,
		LaunchPoll:   time.Hour,
		InstancePoll: time.Hour,
	})

	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(stale); os.IsNotExist(err) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale instance dir still exists: %v", err)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("kept instance dir missing: %v", err)
	}
}

func TestRunLoop_DryLaunchOnce(t *testing.T) {
	jarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("jar-bytes"))
	}))
	t.Cleanup(jarSrv.Close)

	indexBody, _ := json.Marshal(map[string]any{"objects": map[string]any{}})
	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(indexBody)
	}))
	t.Cleanup(indexSrv.Close)

	var completed bool
	pendingServed := false
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/launcher/launch-requests/pending":
			if pendingServed {
				_ = json.NewEncoder(w).Encode(map[string]any{"item": nil})
				return
			}
			pendingServed = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"item": map[string]any{
					"id":          "req-loop-1",
					"status":      "dispatched",
					"instance_id": "inst-1",
					"profile": map[string]any{
						"username":     "Steve",
						"offline_uuid": "00000000-0000-0000-0000-000000000001",
					},
					"instance": map[string]any{
						"id":         "inst-1",
						"name":       "Survival",
						"mc_version": "1.21",
						"loader":     "vanilla",
					},
				},
			})
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/launcher/launch-requests/"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["status"] == "completed" {
				completed = true
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "req-loop-1", "status": body["status"]})
		case r.Method == http.MethodGet && r.URL.Path == "/launcher/update-requests/pending":
			_ = json.NewEncoder(w).Encode(map[string]any{"item": nil})
		case r.Method == http.MethodGet && r.URL.Path == "/launcher/devices/me/instances":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiSrv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go RunLoop(ctx, Config{
		APIBase:          apiSrv.URL,
		DeviceToken:      "device-token",
		DataDir:          filepath.Join(t.TempDir(), "qx"),
		LaunchDryRun:     true,
		SkipJavaDownload: true,
		LaunchPoll:       20 * time.Millisecond,
		InstancePoll:     time.Hour,
		ManifestBuilder: stubManifestBuilder{manifest: &mcmanifest.InstanceLaunchManifest{
			InstanceID: "inst-1",
			MCVersion:  "1.21",
			MainClass:  "Main",
			AssetIndex: mcmanifest.AssetIndexRef{ID: "1.21", URL: indexSrv.URL},
			ClientJar:  mcmanifest.DownloadFile{URL: jarSrv.URL, Sha1: ""},
		}},
	})

	deadline := time.Now().Add(1500 * time.Millisecond)
	for !completed && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if !completed {
		t.Fatal("expected dry-run launch to complete via RunLoop")
	}
}
