//go:build windows

package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWindowsExecutable(t *testing.T) {
	dir := t.TempDir()
	tiny := filepath.Join(dir, "tiny.exe")
	if err := os.WriteFile(tiny, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateWindowsExecutable(tiny); err == nil {
		t.Fatal("expected error for tiny payload")
	}

	bad := filepath.Join(dir, "bad.exe")
	payload := make([]byte, minPEBytes)
	copy(payload, []byte("XX"))
	if err := os.WriteFile(bad, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateWindowsExecutable(bad); err == nil {
		t.Fatal("expected error for non-PE payload")
	}

	good := filepath.Join(dir, "good.exe")
	payload = make([]byte, minPEBytes)
	copy(payload, []byte("MZ"))
	if err := os.WriteFile(good, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateWindowsExecutable(good); err != nil {
		t.Fatalf("valid PE header: %v", err)
	}
}
