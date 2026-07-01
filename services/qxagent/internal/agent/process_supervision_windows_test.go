//go:build windows

package agent

import (
	"os/exec"
	"testing"
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
