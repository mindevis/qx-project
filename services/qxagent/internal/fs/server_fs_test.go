package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWipeWorkDirRemovesAllContent(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "instance-1")
	if err := os.MkdirAll(filepath.Join(workDir, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "server.properties"), []byte("motd=test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "mods", "example.jar"), []byte("mod"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WipeWorkDir(workDir); err != nil {
		t.Fatalf("wipe: %v", err)
	}

	info, err := os.Stat(workDir)
	if err != nil {
		t.Fatalf("work dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("work dir is not a directory")
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty work dir, got %d entries", len(entries))
	}
}

func TestRemoveWorkDirDeletesInstanceFolder(t *testing.T) {
	root := filepath.Join(t.TempDir(), "instances", "gs-1")
	if err := os.MkdirAll(filepath.Join(root, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "server.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorkDir(root); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected instance dir gone, err=%v", err)
	}
}

func TestRemoveWorkDirDeletesUUIDFolderFromNestedPath(t *testing.T) {
	uuidDir := filepath.Join(t.TempDir(), "instances", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	nested := filepath.Join(uuidDir, "world", "datapacks")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uuidDir, "server.properties"), []byte("motd=x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveWorkDir(nested); err != nil {
		t.Fatalf("remove nested: %v", err)
	}
	if _, err := os.Stat(uuidDir); !os.IsNotExist(err) {
		t.Fatalf("expected uuid dir gone, err=%v", err)
	}
}

func TestCopyWorkDirCopiesUUIDFolder(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "instances", "src-gs")
	dest := filepath.Join(root, "instances", "dest-gs")
	if err := os.MkdirAll(filepath.Join(src, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "mods", "mod.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyWorkDir(src, dest); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "mods", "mod.jar")); err != nil {
		t.Fatalf("copied file missing: %v", err)
	}
}

func TestRemoveWorkDirMissingIsOK(t *testing.T) {
	root := filepath.Join(t.TempDir(), "instances", "gs-missing")
	if err := RemoveWorkDir(root); err != nil {
		t.Fatalf("missing dir: %v", err)
	}
}

func TestRemoveWorkDirRejectsOutsideInstances(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveWorkDir(dir); err == nil {
		t.Fatal("expected refusal outside instances")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir should remain: %v", err)
	}
}

func TestWipeWorkDirMissingDirCreatesEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "new-instance")

	if err := WipeWorkDir(workDir); err != nil {
		t.Fatalf("wipe missing dir: %v", err)
	}
	info, err := os.Stat(workDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected created dir: err=%v", err)
	}
}

func TestDeletePathRemovesFileAndFolder(t *testing.T) {
	workDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workDir, "world", "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "eula.txt"), []byte("eula=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "world", "data", "score.dat"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DeletePath(workDir, "eula.txt"); err != nil {
		t.Fatalf("delete file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "eula.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected file gone, err=%v", err)
	}

	if err := DeletePath(workDir, "world"); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, "world")); !os.IsNotExist(err) {
		t.Fatalf("expected folder gone, err=%v", err)
	}
}

func TestDeletePathRejectsWorkDirAndTraversal(t *testing.T) {
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "keep.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"", ".", "/", "..", "../outside"} {
		if err := DeletePath(workDir, path); err == nil {
			t.Fatalf("expected error for path %q", path)
		}
	}
	if _, err := os.Stat(filepath.Join(workDir, "keep.txt")); err != nil {
		t.Fatalf("work dir content should remain: %v", err)
	}
}

func TestMkdirCreatesFolderAndRejectsConflicts(t *testing.T) {
	workDir := t.TempDir()
	if err := Mkdir(workDir, "plugins/configs"); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	info, err := os.Stat(filepath.Join(workDir, "plugins", "configs"))
	if err != nil || !info.IsDir() {
		t.Fatalf("expected folder: err=%v", err)
	}
	if err := Mkdir(workDir, "plugins/configs"); err == nil {
		t.Fatal("expected already exists")
	}
	if err := os.WriteFile(filepath.Join(workDir, "eula.txt"), []byte("eula=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Mkdir(workDir, "eula.txt"); err == nil {
		t.Fatal("expected file conflict")
	}
	for _, path := range []string{"", ".", "/", "..", "../outside"} {
		if err := Mkdir(workDir, path); err == nil {
			t.Fatalf("expected error for path %q", path)
		}
	}
}

func TestWriteFileBytesWritesBinary(t *testing.T) {
	workDir := t.TempDir()
	payload := []byte{0x00, 0x01, 0x02, 0xff}
	if err := WriteFileBytes(workDir, "plugins/demo.jar", payload); err != nil {
		t.Fatalf("write bytes: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(workDir, "plugins", "demo.jar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("bytes: %v", got)
	}
}

func TestListClientModsMissingDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ListClientMods(dir, "forge")
	if err != nil {
		t.Fatalf("list client mods: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(entries))
	}
}
