package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestWriteInputWhenServerNotRunning(t *testing.T) {
	r := &ProcessRunner{}
	var lines []string
	r.SetOutputHandler(func(stream, line, _ string) {
		lines = append(lines, stream+":"+line)
	})
	err := r.WriteInput("list", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(lines) != 1 || lines[0] != "stderr:server process not running" {
		t.Fatalf("got lines %v", lines)
	}
}

func TestWriteInputDryRun(t *testing.T) {
	r := &ProcessRunner{DryRun: true}
	var lines []string
	r.SetOutputHandler(func(stream, line, _ string) {
		lines = append(lines, stream+":"+line)
	})
	if err := r.WriteInput("list", ""); err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected output lines, got %v", lines)
	}
}

func TestStreamLines(t *testing.T) {
	var got []string
	streamLines("stdout", strings.NewReader("line1\nline2\n"), func(stream, line string) {
		got = append(got, stream+":"+line)
	})
	if len(got) != 2 || got[0] != "stdout:line1" {
		t.Fatalf("got %v", got)
	}
}

func TestDryRunStartEmitsOutput(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "server.jar")
	if err := os.WriteFile(jar, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &ProcessRunner{DryRun: true}
	var lines []string
	r.SetOutputHandler(func(stream, line, _ string) {
		lines = append(lines, line)
	})
	if _, err := r.Start(protocol.ServerStartPayload{WorkDir: dir, JarPath: jar}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected dry-run console output")
	}
}
