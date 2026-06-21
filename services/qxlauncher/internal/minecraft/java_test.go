package minecraft

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveJavaBinCustom(t *testing.T) {
	t.Setenv("JAVA_HOME", "")
	got := ResolveJavaBin(`C:\custom\java.exe`)
	if got != `C:\custom\java.exe` {
		t.Fatalf("custom: %q", got)
	}
}

func TestResolveJavaBinFromJavaHome(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.exe"
	}
	javaPath := filepath.Join(bin, name)
	if err := os.WriteFile(javaPath, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JAVA_HOME", dir)
	got := ResolveJavaBin("")
	if got != javaPath {
		t.Fatalf("java home: got %q want %q", got, javaPath)
	}
}

func TestResolveJavaBinFallback(t *testing.T) {
	t.Setenv("JAVA_HOME", "/nonexistent")
	got := ResolveJavaBin("")
	if got == "" {
		t.Fatal("expected non-empty fallback")
	}
}
