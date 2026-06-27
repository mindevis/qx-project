package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadLogTail(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "latest.log")
	if err := os.WriteFile(logPath, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, offset, err := readLogTail(logPath, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("lines: %+v", lines)
	}

	more, nextOffset, err := readLogTail(logPath, offset)
	if err != nil {
		t.Fatalf("read again: %v", err)
	}
	if len(more) != 0 || nextOffset != offset {
		t.Fatalf("expected no new lines, got %+v offset=%d", more, nextOffset)
	}

	if err := os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, _, err = readLogTail(logPath, offset)
	if err != nil {
		t.Fatalf("read appended: %v", err)
	}
	if len(lines) != 1 || lines[0] != "line3" {
		t.Fatalf("appended lines: %+v", lines)
	}
}

func TestFollowServerLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(logDir, "latest.log")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	seen := make(chan string, 4)
	go followServerLog(ctx, dir, func(line string) {
		seen <- line
	})

	time.Sleep(700 * time.Millisecond)
	if err := os.WriteFile(logPath, []byte("boot\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(3 * time.Second)
	for {
		select {
		case line := <-seen:
			if strings.Contains(line, "boot") {
				return
			}
		case <-deadline:
			t.Fatal("timeout waiting for log tail")
		}
	}
}
