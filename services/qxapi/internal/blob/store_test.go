package blob

import (
	"context"
	"io"
	"testing"
)

func TestDirRoundTrip(t *testing.T) {
	store, err := NewDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "resource-uploads/abc", []byte("jar-bytes")); err != nil {
		t.Fatal(err)
	}
	rc, size, err := store.Open(ctx, "resource-uploads/abc")
	if err != nil {
		t.Fatal(err)
	}
	if size != 9 {
		t.Fatalf("size = %d", size)
	}
	got, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "jar-bytes" {
		t.Fatalf("got %q", got)
	}
	if err := store.Delete(ctx, "resource-uploads/abc"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Open(ctx, "resource-uploads/abc"); err == nil {
		t.Fatal("expected missing after delete")
	}
}

func TestSanitizeRejectsTraversal(t *testing.T) {
	if _, err := sanitizeKey("../etc/passwd"); err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenMemory(t *testing.T) {
	store, err := Open(context.Background(), Config{Endpoint: "memory"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(*Memory); !ok {
		t.Fatalf("got %T", store)
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	ep, secure := normalizeEndpoint("https://minio.example:9000", false)
	if ep != "minio.example:9000" || !secure {
		t.Fatalf("https: %q %v", ep, secure)
	}
	ep, secure = normalizeEndpoint("minio:9000", false)
	if ep != "minio:9000" || secure {
		t.Fatalf("plain: %q %v", ep, secure)
	}
}
