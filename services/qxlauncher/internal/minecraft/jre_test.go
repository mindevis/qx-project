package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestEnsureJavaSkipDownload(t *testing.T) {
	dl := NewDownloader(t.TempDir())
	dl.SkipJavaDownload = true
	bin, err := dl.EnsureJava(context.Background(), &mcmanifest.InstanceLaunchManifest{
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

func TestEnsureJavaCustomOverride(t *testing.T) {
	dl := NewDownloader(t.TempDir())
	dl.JavaPath = `C:\custom\java.exe`
	bin, err := dl.EnsureJava(context.Background(), &mcmanifest.InstanceLaunchManifest{})
	if err != nil || bin != `C:\custom\java.exe` {
		t.Fatalf("custom: %v %q", err, bin)
	}
}

func TestEnsureJavaDownloadsPackage(t *testing.T) {
	javaName := "java"
	if runtime.GOOS == "windows" {
		javaName = "java.exe"
	}
	javaBody := []byte("#!/bin/sh\necho java\n")

	mux := http.NewServeMux()
	var serverURL string
	mux.HandleFunc("/catalog.json", func(w http.ResponseWriter, r *http.Request) {
		platform := javaPlatformKey()
		_ = json.NewEncoder(w).Encode(map[string]map[string][]javaRuntimeEntry{
			platform: {
				"java-runtime-delta": {{
					Manifest: struct {
						URL  string `json:"url"`
						Sha1 string `json:"sha1"`
					}{URL: serverURL + "/package.json"},
					Version: struct {
						Name string `json:"name"`
					}{Name: "21.0.7"},
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
							"url":  serverURL + "/java-bin",
							"sha1": "",
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

	oldURL := javaRuntimeCatalogURL
	javaRuntimeCatalogURL = catalogSrv.URL + "/catalog.json"
	t.Cleanup(func() { javaRuntimeCatalogURL = oldURL })

	root := t.TempDir()
	dl := NewDownloader(root)
	dl.HTTPClient = catalogSrv.Client()

	bin, err := dl.EnsureJava(context.Background(), &mcmanifest.InstanceLaunchManifest{
		JavaMajor:     21,
		JavaComponent: "java-runtime-delta",
	})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	want := filepath.Join(root, "java", "java-runtime-delta", "21.0.7", "bin", javaName)
	if bin != want {
		t.Fatalf("bin: got %q want %q", bin, want)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("missing file: %v", err)
	}

	bin2, err := dl.EnsureJava(context.Background(), &mcmanifest.InstanceLaunchManifest{
		JavaMajor:     21,
		JavaComponent: "java-runtime-delta",
	})
	if err != nil || bin2 != bin {
		t.Fatalf("cache: %v %q", err, bin2)
	}
}

func TestComponentForJavaMajor(t *testing.T) {
	cases := map[int]string{
		8:  "jre-legacy",
		16: "java-runtime-alpha",
		17: "java-runtime-gamma",
		21: "java-runtime-delta",
		25: "java-runtime-delta",
	}
	for major, want := range cases {
		if got := componentForJavaMajor(major); got != want {
			t.Fatalf("major %d: got %q want %q", major, got, want)
		}
	}
}

func TestPickJavaDownloadPrefersRaw(t *testing.T) {
	got := pickJavaDownload(map[string]mcmanifest.DownloadFile{
		"lzma": {URL: "lzma"},
		"raw":  {URL: "raw"},
	})
	if got.URL != "raw" {
		t.Fatalf("pick: %+v", got)
	}
}

// TestIntegrationJavaOnPath runs `java -version` when a JVM is available (skipped with -short).
func TestIntegrationJavaOnPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping JVM smoke test in short mode")
	}

	dl := NewDownloader(t.TempDir())
	dl.SkipJavaDownload = true
	bin, err := dl.EnsureJava(context.Background(), &mcmanifest.InstanceLaunchManifest{
		JavaMajor: 21,
	})
	if err != nil {
		t.Fatalf("ensure java: %v", err)
	}
	if err := runJavaVersion(context.Background(), bin); err != nil {
		t.Fatalf("java -version: %v", err)
	}
}

func runJavaVersion(ctx context.Context, javaBin string) error {
	cmd := exec.CommandContext(ctx, javaBin, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
