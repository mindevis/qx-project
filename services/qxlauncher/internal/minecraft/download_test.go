package minecraft

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestIsRetryableDownloadError(t *testing.T) {
	cases := map[string]bool{
		"Get \"https://example.com\": net/http: TLS handshake timeout": true,
		"connection reset by peer":  true,
		"download failed: status 404": false,
	}
	for msg, want := range cases {
		if got := isRetryableDownloadError(fmt.Errorf("%s", msg)); got != want {
			t.Fatalf("%q: got %v want %v", msg, got, want)
		}
	}
}

func TestDownloadWithRetry(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "artifact.jar")
	dl := NewDownloader(dir)

	err := dl.downloadWithRetry(context.Background(), srv.URL, dest)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("attempts: %d want 3", attempts.Load())
	}
	if b, err := os.ReadFile(dest); err != nil || string(b) != "ok" {
		t.Fatalf("file: %q err=%v", b, err)
	}
}

func TestDownloadIfNeededUsesCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not download when sha1 matches")
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "cached.jar")
	content := []byte("cached-bytes")
	if err := os.WriteFile(dest, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha1Sum(content)
	dl := NewDownloader(dir)
	if err := dl.downloadIfNeeded(context.Background(), srv.URL, dest, fmt.Sprintf("%x", sum)); err != nil {
		t.Fatalf("downloadIfNeeded: %v", err)
	}
}

func TestDownloadWithRetryNoRetryOn404(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	dest := filepath.Join(dir, "missing.jar")
	dl := NewDownloader(dir)

	err := dl.downloadWithRetry(context.Background(), srv.URL, dest)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected 404 error, got %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts: %d want 1", attempts.Load())
	}
}
