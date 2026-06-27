package mojangjava_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/pkg/mojangjava"
)

func TestEnsureForManifestSkipDownload(t *testing.T) {
	m := &mojangjava.Manager{RootDir: t.TempDir(), SkipDownload: true}
	bin, err := m.EnsureForManifest(context.Background(), &mcmanifest.InstanceLaunchManifest{
		JavaMajor:     21,
		JavaComponent: "java-runtime-delta",
	})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if bin == "" {
		t.Fatal("expected java bin")
	}
}

func TestEnsureForManifestCustomOverride(t *testing.T) {
	m := &mojangjava.Manager{RootDir: t.TempDir(), JavaPath: `C:\custom\java.exe`}
	bin, err := m.EnsureForManifest(context.Background(), &mcmanifest.InstanceLaunchManifest{})
	if err != nil || bin != `C:\custom\java.exe` {
		t.Fatalf("custom: %v %q", err, bin)
	}
}

func TestEnsureForManifestDownloadsPackage(t *testing.T) {
	javaName := "java"
	if runtime.GOOS == "windows" {
		javaName = "java.exe"
	}
	javaBody := []byte("#!/bin/sh\necho java\n")

	mux := http.NewServeMux()
	var serverURL string
	mux.HandleFunc("/catalog.json", func(w http.ResponseWriter, r *http.Request) {
		platform := mojangjava.PlatformKey()
		_ = json.NewEncoder(w).Encode(map[string]map[string][]map[string]any{
			platform: {
				"java-runtime-delta": {{
					"manifest": map[string]string{
						"url": serverURL + "/package.json",
					},
					"version": map[string]string{
						"name": "21.0.7",
					},
				}},
			},
		})
	})
	mux.HandleFunc("/package.json", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"files": map[string]any{
				"bin/" + javaName: map[string]any{
					"type":       "file",
					"executable": true,
					"downloads": map[string]any{
						"raw": map[string]any{
							"url": serverURL + "/java-bin",
						},
					},
				},
			},
		})
	})
	mux.HandleFunc("/java-bin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(javaBody)
	})
	catalogSrv := httptest.NewServer(mux)
	t.Cleanup(catalogSrv.Close)
	serverURL = catalogSrv.URL

	root := t.TempDir()
	m := &mojangjava.Manager{
		RootDir:    root,
		HTTPClient: catalogSrv.Client(),
		CatalogURL: catalogSrv.URL + "/catalog.json",
	}

	bin, err := m.EnsureForManifest(context.Background(), &mcmanifest.InstanceLaunchManifest{
		JavaMajor:     21,
		JavaComponent: "java-runtime-delta",
	})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	want := filepath.Join(root, "java-runtime-delta", "21.0.7", "bin", javaName)
	if bin != want {
		t.Fatalf("bin: got %q want %q", bin, want)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("missing file: %v", err)
	}

	bin2, err := m.EnsureForManifest(context.Background(), &mcmanifest.InstanceLaunchManifest{
		JavaMajor:     21,
		JavaComponent: "java-runtime-delta",
	})
	if err != nil || bin2 != bin {
		t.Fatalf("cache: %v %q", err, bin2)
	}
}

func TestComponentForMajor(t *testing.T) {
	cases := map[int]string{
		8:  "jre-legacy",
		16: "java-runtime-alpha",
		17: "java-runtime-gamma",
		21: "java-runtime-delta",
		25: "java-runtime-delta",
	}
	for major, want := range cases {
		if got := mojangjava.ComponentForMajor(major); got != want {
			t.Fatalf("major %d: got %q want %q", major, got, want)
		}
	}
}
