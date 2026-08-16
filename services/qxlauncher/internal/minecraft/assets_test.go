package minecraft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestEnsureAssets(t *testing.T) {
	dir := t.TempDir()
	objectBody := []byte("asset-bytes")
	hash := "0123456789abcdef0123456789abcdef01234567"

	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/01/"+hash {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(objectBody)
	}))
	t.Cleanup(cdnSrv.Close)

	indexBody, _ := json.Marshal(assetIndexFile{
		Objects: map[string]assetObject{
			"minecraft/lang/en_us.json": {Hash: hash, Size: int64(len(objectBody))},
		},
	})
	indexSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(indexBody)
	}))
	t.Cleanup(indexSrv.Close)

	manifest := &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		AssetIndex: mcmanifest.AssetIndexRef{
			ID:  "test-index",
			URL: indexSrv.URL,
		},
	}

	dl := NewDownloader(dir)
	dl.AssetsCDN = cdnSrv.URL
	assetsDir, err := dl.EnsureAssets(context.Background(), manifest)
	if err != nil {
		t.Fatalf("ensure assets: %v", err)
	}
	wantAssets, err := dl.InstanceAssetsDir("inst-1")
	if err != nil {
		t.Fatal(err)
	}
	if assetsDir != wantAssets {
		t.Fatalf("assets dir: got %s want %s", assetsDir, wantAssets)
	}
	objectPath := filepath.Join(assetsDir, "objects", "01", hash)
	if b, err := os.ReadFile(objectPath); err != nil || string(b) != string(objectBody) {
		t.Fatalf("object: err=%v body=%q", err, b)
	}
}

func TestEnsureAssetsMissingIndex(t *testing.T) {
	dl := NewDownloader(t.TempDir())
	_, err := dl.EnsureAssets(context.Background(), &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "x"},
	})
	if err == nil {
		t.Fatal("expected error without index url")
	}
}
