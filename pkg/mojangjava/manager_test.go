package mojangjava_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestEnsureForRuntimeRequiresRoot(t *testing.T) {
	m := &mojangjava.Manager{}
	_, err := m.EnsureForRuntime(context.Background(), "java-runtime-delta", 25)
	if err == nil {
		t.Fatal("expected java root error")
	}
}

func TestEnsureForRuntimeDownloadsTemurin(t *testing.T) {
	javaName := "java"
	if runtime.GOOS == "windows" {
		javaName = "java.exe"
	}
	archive, err := buildTemurinTestArchive(javaName)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}

	var sawUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(archive)
	}))
	t.Cleanup(srv.Close)

	root := t.TempDir()
	m := &mojangjava.Manager{
		RootDir:    root,
		HTTPClient: srv.Client(),
		TemurinURL: srv.URL + "/OpenJDK25U-jdk_x64_linux_hotspot_25.tgz",
	}
	bin, err := m.EnsureForRuntime(context.Background(), "java-runtime-delta", 25)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	want := filepath.Join(root, "temurin", "25", "jdk-25", "bin", javaName)
	if bin != want {
		t.Fatalf("bin: got %q want %q", bin, want)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("missing file: %v", err)
	}
	if !strings.Contains(sawUA, "QXProject") {
		t.Fatalf("user-agent: %q", sawUA)
	}

	bin2, err := m.EnsureForRuntime(context.Background(), "java-runtime-delta", 25)
	if err != nil || bin2 != bin {
		t.Fatalf("cache: %v %q", err, bin2)
	}
}

func TestEnsureForRuntimeFindsTemurinMacLayout(t *testing.T) {
	javaName := "java"
	if runtime.GOOS == "windows" {
		javaName = "java.exe"
	}
	root := t.TempDir()
	binPath := filepath.Join(root, "temurin", "25", "jdk-25.0.1", "Contents", "Home", "bin", javaName)
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := &mojangjava.Manager{RootDir: root, TemurinURL: "http://127.0.0.1:1/missing"}
	bin, err := m.EnsureForRuntime(context.Background(), "", 25)
	if err != nil || bin != binPath {
		t.Fatalf("got %q err %v", bin, err)
	}
}

func buildTemurinTestArchive(javaName string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("#!/bin/sh\necho java\n")
	hdr := &tar.Header{
		Name: "jdk-25/bin/" + javaName,
		Mode: 0o755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(body); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
