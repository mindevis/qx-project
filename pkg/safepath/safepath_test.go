package safepath

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPartPath(t *testing.T) {
	dir := t.TempDir()
	path, err := Join(dir, "file.jar")
	if err != nil {
		t.Fatal(err)
	}
	part, err := PartPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if part != path+".part" {
		t.Fatalf("part=%q", part)
	}
}

func TestWriteStreamAtomic(t *testing.T) {
	dir := t.TempDir()
	dest, err := Join(dir, "download.bin")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteStreamAtomic(dest, strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "hello" {
		t.Fatalf("data=%q err=%v", data, err)
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
