package agent

import (
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestWriteInputDryRun(t *testing.T) {
	r := &ProcessRunner{DryRun: true}
	var lines []string
	r.SetOutputHandler(func(stream, line string) {
		lines = append(lines, stream+":"+line)
	})
	if err := r.WriteInput("list"); err != nil {
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
	r := &ProcessRunner{DryRun: true}
	var lines []string
	r.SetOutputHandler(func(stream, line string) {
		lines = append(lines, line)
	})
	if _, err := r.Start(protocol.ServerStartPayload{JarPath: "/tmp/server.jar"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected dry-run console output")
	}
}
