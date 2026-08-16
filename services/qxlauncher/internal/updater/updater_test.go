package updater

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveURL(t *testing.T) {
	got := ResolveURL("https://mc.qx-dev.ru/api/v1", "/downloads/qx-launcher.exe")
	want := "https://mc.qx-dev.ru/downloads/qx-launcher.exe"
	if got != want {
		t.Fatalf("ResolveURL: got %q want %q", got, want)
	}
	abs := "https://cdn.example/qx-launcher.exe"
	if got := ResolveURL("https://mc.qx-dev.ru/api/v1", abs); got != abs {
		t.Fatalf("ResolveURL absolute: got %q", got)
	}
}

func TestApplyUnsupportedOS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows test")
	}
	err := Apply(context.Background(), "http://example.com/qx-launcher.exe", "qx-launcher.exe", nil)
	if err == nil {
		t.Fatal("expected error on non-windows")
	}
}

func TestBackupPath(t *testing.T) {
	got := backupPath(`C:\Apps\qx-launcher.exe`)
	want := `C:\Apps\qx-launcher.exe.prev`
	if got != want {
		t.Fatalf("backupPath: got %q want %q", got, want)
	}
}

func TestStagingDirUsesLocalAppData(t *testing.T) {
	local := `C:\Users\test\AppData\Local`
	t.Setenv("LOCALAPPDATA", local)
	got := stagingDir()
	want := filepath.Join(local, "QXLauncher", "updates")
	if got != want {
		t.Fatalf("stagingDir: got %q want %q", got, want)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	if err := os.WriteFile(src, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("copyFile: got %q", data)
	}
}
