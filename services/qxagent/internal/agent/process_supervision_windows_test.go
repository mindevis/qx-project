//go:build windows

package agent

import (
	"os/exec"
	"testing"
	"time"
)

func TestProcessRunnerDetectsDeadProcess(t *testing.T) {
	r := &ProcessRunner{}
	cmd := exec.Command("powershell", "-Command", "Start-Sleep -Seconds 30")
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

func TestProcessRunnerStopKillsManagedProcess(t *testing.T) {
	cmd := exec.Command("powershell", "-Command", "Start-Sleep -Seconds 30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	r := &ProcessRunner{}
	r.mu.Lock()
	r.cmd = cmd
	r.mu.Unlock()
	go r.watchManagedProcess(cmd, "")

	if _, err := r.Stop(false, 2*time.Second); err != nil {
		t.Fatalf("stop: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("process did not exit")
	}
}
