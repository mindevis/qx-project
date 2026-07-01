//go:build !windows

package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestProcessRunnerDetectsDeadProcess(t *testing.T) {
	r := &ProcessRunner{}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	r.mu.Lock()
	r.cmd = cmd
	r.mu.Unlock()
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	r.mu.Lock()
	r.cleanupDeadProcessLocked()
	alive := r.cmd != nil
	r.mu.Unlock()
	if alive {
		t.Fatal("expected dead managed process to be cleared")
	}
}

func TestProcessRunnerEmitsCrashedStatus(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(logDir, "latest.log")
	if err := os.WriteFile(logPath, []byte("fatal boot error\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	statusCh := make(chan protocol.ServerStatusPayload, 1)
	r := &ProcessRunner{}
	r.SetStatusHandler(func(payload protocol.ServerStatusPayload) {
		statusCh <- payload
	})

	cmd := exec.Command("sh", "-c", "exit 7")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	r.mu.Lock()
	r.cmd = cmd
	r.gameServerID = "gs-crash"
	r.managedWorkDir = dir
	r.mu.Unlock()

	go r.watchManagedProcess(cmd, dir)

	select {
	case payload := <-statusCh:
		if payload.Status != protocol.ServerStatusCrashed {
			t.Fatalf("status: %+v", payload)
		}
		if payload.GameServerID != "gs-crash" {
			t.Fatalf("game server id: %s", payload.GameServerID)
		}
		if payload.Message == "" {
			t.Fatal("expected crash message")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for crash status")
	}
}
