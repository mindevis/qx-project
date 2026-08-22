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
	r.adoptCmdForTest(cmd, "", "")
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	r.mu.Lock()
	r.cleanupDeadProcessLocked()
	alive := r.hasManagedCmdLocked()
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
	r.adoptCmdForTest(cmd, dir, "gs-crash")

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

func TestProcessRunnerStopKillsShellChild(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sh", "-c", "sleep 60")
	cmd.Dir = dir
	configureProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	r := &ProcessRunner{}
	r.adoptCmdForTest(cmd, dir, "")
	go r.watchManagedProcess(cmd, dir)

	if _, err := r.Stop(false, 2*time.Second); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitExited(t, cmd, 3*time.Second)
	if pids := runningWorkDirPIDs(dir); len(pids) > 0 {
		t.Fatalf("child processes still running: %v", pids)
	}
}

func TestProcessRunnerStopKillsOrphanInWorkDir(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sleep", "60")
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	r := &ProcessRunner{}
	if _, err := r.StopTarget(false, 2*time.Second, dir); err != nil {
		t.Fatalf("stop: %v", err)
	}
	waitExited(t, cmd, 3*time.Second)
	if processRunning(cmd.Process.Pid) {
		t.Fatal("orphan process still running")
	}
}

func waitExited(t *testing.T, cmd *exec.Cmd, timeout time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("process did not exit")
	}
}

func runningWorkDirPIDs(workDir string) []int {
	var alive []int
	for _, pid := range pidsInWorkDir(workDir) {
		if processRunning(pid) {
			alive = append(alive, pid)
		}
	}
	return alive
}
