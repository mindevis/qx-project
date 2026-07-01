package agent

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

func (r *ProcessRunner) SetStatusHandler(fn func(protocol.ServerStatusPayload)) {
	r.mu.Lock()
	r.onStatus = fn
	r.mu.Unlock()
}

func (r *ProcessRunner) CurrentStatus() protocol.ServerStatusPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentStatusLocked()
}

func (r *ProcessRunner) currentStatusLocked() protocol.ServerStatusPayload {
	if r.cmd == nil || r.cmd.Process == nil {
		return protocol.ServerStatusPayload{Status: protocol.ServerStatusStopped}
	}
	pid := r.cmd.Process.Pid
	if !processAlive(pid) {
		return protocol.ServerStatusPayload{Status: protocol.ServerStatusStopped}
	}
	return protocol.ServerStatusPayload{
		Status:       protocol.ServerStatusRunning,
		PID:          pid,
		GameServerID: r.gameServerID,
	}
}

func (r *ProcessRunner) clearManagedProcessLocked() {
	r.cmd = nil
	r.gameServerID = ""
	r.stoppingGracefully = false
	r.closePipesLocked()
	r.stopLogFollowLocked()
}

func (r *ProcessRunner) cleanupDeadProcessLocked() {
	if r.cmd == nil || r.cmd.Process == nil {
		return
	}
	if processAlive(r.cmd.Process.Pid) {
		return
	}
	r.clearManagedProcessLocked()
}

func (r *ProcessRunner) watchManagedProcess(cmd *exec.Cmd, workDir string) {
	err := cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	} else if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	r.mu.Lock()
	stopping := r.stoppingGracefully
	gameServerID := r.gameServerID
	onStatus := r.onStatus
	if r.cmd == cmd {
		r.clearManagedProcessLocked()
	}
	r.mu.Unlock()

	if onStatus == nil {
		return
	}
	if stopping {
		onStatus(protocol.ServerStatusPayload{
			GameServerID: gameServerID,
			Status:       protocol.ServerStatusStopped,
			ExitCode:     &exitCode,
		})
		return
	}
	message := crashMessage(workDir, exitCode)
	onStatus(protocol.ServerStatusPayload{
		GameServerID: gameServerID,
		Status:       protocol.ServerStatusCrashed,
		ExitCode:     &exitCode,
		Message:      message,
	})
}

func crashMessage(workDir string, exitCode int) string {
	lines, err := readRecentLogLines(workDirLogPath(workDir), 12)
	if err != nil || len(lines) == 0 {
		return fmt.Sprintf("minecraft server exited unexpectedly (code %d)", exitCode)
	}
	tail := strings.Join(lines, "\n")
	if len(tail) > 2000 {
		tail = tail[len(tail)-2000:]
	}
	return fmt.Sprintf("minecraft server exited unexpectedly (code %d)\n%s", exitCode, tail)
}

func workDirLogPath(workDir string) string {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return ""
	}
	path, err := safepath.Join(workDir, "logs", "latest.log")
	if err != nil {
		return ""
	}
	return path
}
