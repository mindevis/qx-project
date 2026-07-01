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
	want := filepath.Join(dir, "instances", "inst-1", "libraries", "com", "test", "demo", "1.0", "demo-1.0.jar")
	if paths[0] != want {
		t.Fatalf("lib path = %q, want %q", paths[0], want)
	}
}

func TestEnsureNativesExtract(t *testing.T) {
	classifier, nativeName := nativeClassifier(), ""
	switch classifier {
	case "natives-windows":
		nativeName = "foo.dll"
	case "natives-macos":
		nativeName = "foo.dylib"
	default:
		nativeName = "foo.so"
	}

	dir := t.TempDir()
	nativeJar := filepath.Join(dir, "native.jar")
	f, err := os.Create(nativeJar)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create(nativeName)
	_, _ = w.Write([]byte("native"))
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
						classifier: {URL: srv.URL, Sha1: ""},
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
	if _, err := os.Stat(filepath.Join(nativesDir, nativeName)); err != nil {
		t.Fatalf("missing native %s: %v", nativeName, err)
	}
}

func TestEnsureNativesNamedArtifact(t *testing.T) {
	classifier := nativeClassifier()
	dir := t.TempDir()
	nativeJar := filepath.Join(dir, "native.jar")
	f, err := os.Create(nativeJar)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("lwjgl.dll")
	_, _ = w.Write([]byte("native"))
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
				Name: "org.lwjgl:lwjgl:3.3.1:" + classifier,
				Downloads: &mcmanifest.LibraryDownloads{
					Artifact: &mcmanifest.DownloadFile{URL: srv.URL, Sha1: ""},
				},
			},
		},
	}
	dl := NewDownloader(dir)
	nativesDir, err := dl.EnsureNatives(context.Background(), manifest)
	if err != nil {
		t.Fatalf("natives: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nativesDir, "lwjgl.dll")); err != nil {
		t.Fatalf("missing lwjgl.dll: %v", err)
	}
}

func TestIsNamedNativeLibrary(t *testing.T) {
	if !isNamedNativeLibrary("org.lwjgl:lwjgl:3.3.1:natives-windows") {
		t.Fatal("expected native library name")
	}
	if isNamedNativeLibrary("org.lwjgl:lwjgl:3.3.1") {
		t.Fatal("expected non-native library name")
	}
}

func TestIsNativeBinary(t *testing.T) {
	if !isNativeBinary("x.dll") || isNativeBinary("readme.txt") {
		t.Fatal("native binary filter")
	}
}
