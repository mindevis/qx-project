package minecraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchFailureHintLWJGL(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "launch.log")
	errText := "java.lang.UnsatisfiedLinkError: Failed to locate library: lwjgl.dll"
	if err := os.WriteFile(logPath, []byte(errText), 0o644); err != nil {
		t.Fatal(err)
	}
	hint := LaunchFailureHint(logPath, dir)
	lower := strings.ToLower(hint)
	if hint == "" || !strings.Contains(lower, "lwjgl") || !strings.Contains(lower, "natives") {
		t.Fatalf("hint = %q", hint)
	}
}
