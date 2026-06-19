package minecraft

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestEnsureLibraries(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("lib-bytes"))
	}))
	t.Cleanup(srv.Close)

	manifest := &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		Libraries: []mcmanifest.Library{
			{
				Name: "com.test:demo:1.0",
				Downloads: &mcmanifest.LibraryDownloads{
					Artifact: &mcmanifest.DownloadFile{URL: srv.URL, Sha1: ""},
				},
			},
		},
	}
	dl := NewDownloader(dir)
	paths, err := dl.EnsureLibraries(context.Background(), manifest)
	if err != nil || len(paths) != 1 {
		t.Fatalf("libs: err=%v paths=%v", err, paths)
	}
}

func TestEnsureNativesExtract(t *testing.T) {
	dir := t.TempDir()
	nativeJar := filepath.Join(dir, "native.jar")
	f, err := os.Create(nativeJar)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("foo.dll")
	_, _ = w.Write([]byte("dll"))
	_ = zw.Close()
	_ = f.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := os.ReadFile(nativeJar)
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)

	manifest := &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		Libraries: []mcmanifest.Library{
			{
				Name: "org.lwjgl:lwjgl:3.3.3",
				Downloads: &mcmanifest.LibraryDownloads{
					Classifiers: map[string]mcmanifest.DownloadFile{
						"natives-windows": {URL: srv.URL, Sha1: ""},
					},
				},
			},
		},
	}
	dl := NewDownloader(dir)
	nativesDir, err := dl.EnsureNatives(context.Background(), manifest)
	if err != nil {
		t.Fatalf("natives: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nativesDir, "foo.dll")); err != nil {
		t.Fatalf("missing dll: %v", err)
	}
}

func TestIsNativeBinary(t *testing.T) {
	if !isNativeBinary("x.dll") || isNativeBinary("readme.txt") {
		t.Fatal("native binary filter")
	}
}
