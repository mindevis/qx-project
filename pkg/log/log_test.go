package log

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"WARNING", slog.LevelWarn},
		{"warn", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}
	for _, tc := range tests {
		if got := ParseLevel(tc.in); got != tc.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSetupTextAndJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := Setup(Options{Level: "DEBUG", Format: "text", Output: &buf})
	logger.Info("hello", "key", "value")
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("text log: %q", buf.String())
	}

	buf.Reset()
	Setup(Options{Level: "ERROR", Format: "json", Output: &buf})
	slog.Info("hidden")
	slog.Error("visible", "code", 1)
	if !strings.Contains(buf.String(), `"visible"`) || strings.Contains(buf.String(), "hidden") {
		t.Fatalf("json log: %q", buf.String())
	}
}

func TestSetupFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "ERROR")
	t.Setenv("LOG_FORMAT", "json")
	var buf bytes.Buffer
	old := slog.Default()
	t.Cleanup(func() { slog.SetDefault(old) })

	Setup(Options{
		Level:  os.Getenv("LOG_LEVEL"),
		Format: os.Getenv("LOG_FORMAT"),
		Output: &buf,
	})
	slog.Error("from env")
	if !strings.Contains(buf.String(), "from env") {
		t.Fatalf("env setup: %q", buf.String())
	}
	if SetupFromEnv() == nil {
		t.Fatal("expected logger")
	}
}

func TestSetupNilOutputUsesStdout(t *testing.T) {
	logger := Setup(Options{Level: "INFO"})
	if logger == nil {
		t.Fatal("expected logger")
	}
}
