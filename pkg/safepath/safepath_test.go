package safepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJoinRelBlocksTraversal(t *testing.T) {
	dir := t.TempDir()
	if _, err := JoinRel(dir, "../etc/passwd"); err == nil {
		t.Fatal("expected traversal error")
	}
}

func TestJoinFixedSegments(t *testing.T) {
	dir := t.TempDir()
	path, err := Join(dir, "logs", "latest.log")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "logs", "latest.log")
	if path != want {
		t.Fatalf("got %q want %q", path, want)
	}
}

func TestZipEntryBase(t *testing.T) {
	if _, err := ZipEntryBase("../../etc/passwd"); err == nil {
		t.Fatal("expected zip traversal error")
	}
	base, err := ZipEntryBase("META-INF/native/linux/libfoo.so")
	if err != nil || base != "libfoo.so" {
		t.Fatalf("base=%q err=%v", base, err)
	}
}

func TestResolveUnderAbsolute(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "server.jar")
	if err := os.WriteFile(child, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveUnder(dir, child)
	if err != nil || got != child {
		t.Fatalf("ResolveUnder: got %q err=%v", got, err)
	}
}
