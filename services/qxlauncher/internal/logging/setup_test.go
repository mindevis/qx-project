package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	qxlog "github.com/qxproject/qx/pkg/log"
)

func TestSetupWritesLogFile(t *testing.T) {
	dir := t.TempDir()
	path := Setup(dir, qxlog.Options{Level: "INFO", Format: "text"})
	if path == "" {
		t.Fatal("expected log path")
	}
	want := filepath.Join(dir, "logs", "qxlauncher.log")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "qxlauncher log file enabled") {
		t.Fatalf("log content: %q", string(data))
	}
}
