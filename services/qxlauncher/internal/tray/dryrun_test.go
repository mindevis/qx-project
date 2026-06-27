package tray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/minecraft"
)

type dryRunHarness struct {
	t          *testing.T
	jarURL     string
	indexURL   string
	api        *apiclient.Client
	lastStatus string
}

func newDryRunHarness(t *testing.T) *dryRunHarness {
	t.Helper()
	jarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("jar-bytes"))
	}))
	t.Cleanup(jarSrv.Close)

	indexBody, _ := json.Marshal(map[string]any{"objects": map[string]any{}})
	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(indexBody)
	}))
	t.Cleanup(indexSrv.Close)

	h := &dryRunHarness{t: t, jarURL: jarSrv.URL, indexURL: indexSrv.URL}
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["status"].(string); ok {
			h.lastStatus = s
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "req", "status": h.lastStatus})
	}))
	t.Cleanup(apiSrv.Close)
	h.api = apiclient.New(apiSrv.URL, "token")
	return h
}

func (h *dryRunHarness) run(t *testing.T, item *apiclient.LaunchRequestItem) {
	t.Helper()
	dl := minecraft.NewDownloader(filepath.Join(t.TempDir(), "data"))
	dl.SkipJavaDownload = true
	executeLaunch(context.Background(), h.api, dl, Config{LaunchDryRun: true}, item)
	if h.lastStatus != "completed" {
		t.Fatalf("expected completed, got %q", h.lastStatus)
	}
}

func TestExecuteLaunchDryRunFabric(t *testing.T) {
	h := newDryRunHarness(t)
	h.run(t, &apiclient.LaunchRequestItem{
		ID: "req-fabric",
		Profile: &apiclient.OfflineProfile{
			Username:    "FabricTest",
			OfflineUUID: "00000000-0000-0000-0000-000000000001",
		},
		Manifest: &mcmanifest.InstanceLaunchManifest{
			InstanceID:    "inst-fabric",
			MCVersion:     "1.21.1",
			Loader:        mcmanifest.LoaderFabric,
			LoaderVersion: "0.19.3",
			VersionID:     "fabric-loader-0.19.3-1.21.1",
			MainClass:     "net.fabricmc.loader.impl.launch.knot.KnotClient",
			AssetIndex:    mcmanifest.AssetIndexRef{ID: "1.21.1", URL: h.indexURL},
			ClientJar:     mcmanifest.DownloadFile{URL: h.jarURL, Sha1: ""},
			JVMArguments:  []string{"-Xmx2G"},
			GameArguments: []string{
				"--username", "${auth_player_name}",
				"--gameDir", "${game_directory}",
			},
		},
	})
}

func TestExecuteLaunchDryRunQuilt(t *testing.T) {
	h := newDryRunHarness(t)
	h.run(t, &apiclient.LaunchRequestItem{
		ID: "req-quilt",
		Profile: &apiclient.OfflineProfile{
			Username:    "QuiltTest",
			OfflineUUID: "00000000-0000-0000-0000-000000000001",
		},
		Manifest: &mcmanifest.InstanceLaunchManifest{
			InstanceID:    "inst-quilt",
			MCVersion:     "1.21.1",
			Loader:        mcmanifest.LoaderQuilt,
			LoaderVersion: "0.28.1",
			VersionID:     "quilt-loader-0.28.1-1.21.1",
			MainClass:     "org.quiltmc.loader.impl.launch.knot.KnotClient",
			AssetIndex:    mcmanifest.AssetIndexRef{ID: "1.21.1", URL: h.indexURL},
			ClientJar:     mcmanifest.DownloadFile{URL: h.jarURL, Sha1: ""},
			JVMArguments:  []string{"-Xmx2G"},
			GameArguments: []string{
				"--username", "${auth_player_name}",
				"--gameDir", "${game_directory}",
			},
		},
	})
}
