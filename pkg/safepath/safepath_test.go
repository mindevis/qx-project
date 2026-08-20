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
	if part.String() != path+".part" {
		t.Fatalf("part=%q", part)
	}
}

func TestVettedAbsRejectsRelativeAndDotDot(t *testing.T) {
	if _, err := VettedAbs("relative/file.jar"); err == nil {
		t.Fatal("expected relative path error")
	}
	dir := t.TempDir()
	escaped := dir + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "outside.jar"
	if _, err := VettedAbs(escaped); err == nil {
		t.Fatal("expected unclean path error")
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

func TestRemoveInstancesChildDeletesUUIDDir(t *testing.T) {
	instances := filepath.Join(t.TempDir(), "instances")
	uuidDir := filepath.Join(instances, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err := os.MkdirAll(filepath.Join(uuidDir, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uuidDir, "server.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveInstancesChild(uuidDir); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(uuidDir); !os.IsNotExist(err) {
		t.Fatalf("uuid dir still exists: %v", err)
	}
	if _, err := os.Stat(instances); err != nil {
		t.Fatalf("instances parent missing: %v", err)
	}
}

func TestRemoveInstancesChildFromNestedPath(t *testing.T) {
	instances := filepath.Join(t.TempDir(), "instances")
	uuidDir := filepath.Join(instances, "11111111-2222-3333-4444-555555555555")
	nested := filepath.Join(uuidDir, "world", "datapacks")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RemoveInstancesChild(nested); err != nil {
		t.Fatalf("remove nested: %v", err)
	}
	if _, err := os.Stat(uuidDir); !os.IsNotExist(err) {
		t.Fatalf("uuid dir still exists: %v", err)
	}
}

func TestRemoveInstancesChildMissingIsOK(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "instances", "00000000-0000-0000-0000-000000000000")
	if err := RemoveInstancesChild(missing); err != nil {
		t.Fatalf("missing: %v", err)
	}
}

func TestRemoveInstancesChildRejectsOutsideInstances(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveInstancesChild(dir); err == nil {
		t.Fatal("expected refusal outside instances")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir should remain: %v", err)
	}
}

func TestRemoveInstancesChildClearsReadOnlyFiles(t *testing.T) {
	uuidDir := filepath.Join(t.TempDir(), "instances", "readonly-id")
	if err := os.MkdirAll(uuidDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(uuidDir, "locked.dat")
	if err := os.WriteFile(file, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := RemoveInstancesChild(uuidDir); err != nil {
		t.Fatalf("remove readonly: %v", err)
	}
	if _, err := os.Stat(uuidDir); !os.IsNotExist(err) {
		t.Fatalf("uuid dir still exists: %v", err)
	}
}

func TestCopyInstancesChild(t *testing.T) {
	instances := filepath.Join(t.TempDir(), "instances")
	src := filepath.Join(instances, "src-id")
	dest := filepath.Join(instances, "dest-id")
	if err := os.MkdirAll(filepath.Join(src, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "mods", "a.jar"), []byte("mod"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "server.properties"), []byte("motd=hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "session.lock"), []byte("lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyInstancesChild(src, dest); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "mods", "a.jar")); err != nil {
		t.Fatalf("copied jar missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "session.lock")); !os.IsNotExist(err) {
		t.Fatal("session.lock should not be copied")
	}
	if err := CopyInstancesChild(src, src); err == nil {
		t.Fatal("expected same-dir error")
	}
}
