package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestAttachConsoleReadsRecentLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "latest.log"), []byte("line-a\nline-b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var lines []string
	r := &ProcessRunner{}
	r.SetOutputHandler(func(stream, line string) {
		lines = append(lines, stream+":"+line)
	})
	r.AttachConsole(protocol.ConsoleAttachPayload{
		GameServerID: "gs-1",
		WorkDir:      dir,
	})

	if len(lines) < 2 || lines[0] != "log:line-a" || lines[1] != "log:line-b" {
		t.Fatalf("lines: %+v", lines)
	}
}
