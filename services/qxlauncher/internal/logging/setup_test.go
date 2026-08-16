package logging

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qxlog "github.com/qxproject/qx/pkg/log"
)

func TestSetupWritesLogFile(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(Close)
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

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("no console")
}

func TestSetupWritesWhenConsoleFails(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(Close)
	path := Setup(dir, qxlog.Options{Level: "INFO", Format: "text", Output: failWriter{}})
	if path == "" {
		t.Fatal("expected log path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "qxlauncher log file enabled") {
		t.Fatalf("expected file log despite console error, got %q", data)
	}
}

func TestRotateCreatesBackup(t *testing.T) {
	prevBytes, prevKeep := rotateBytes, rotateKeep
	rotateBytes = 64
	rotateKeep = 2
	t.Cleanup(func() {
		rotateBytes = prevBytes
		rotateKeep = prevKeep
	})

	dir := t.TempDir()
	t.Cleanup(Close)
	path := Setup(dir, qxlog.Options{Level: "INFO", Format: "text", Output: failWriter{}})
	if path == "" {
		t.Fatal("expected log path")
	}
	line := strings.Repeat("x", 80)
	for i := 0; i < 4; i++ {
		slog.Info(line)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected rotated log: %v", err)
	}
}
