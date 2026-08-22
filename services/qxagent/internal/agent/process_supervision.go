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
	statuses := r.CurrentStatuses()
	return statuses[0]
}

func (r *ProcessRunner) CurrentStatuses() []protocol.ServerStatusPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupDeadProcessLocked()
	var out []protocol.ServerStatusPayload
	for _, inst := range r.instances {
		if inst.isAlive(r.DryRun) {
			out = append(out, protocol.ServerStatusPayload{
				Status:       protocol.ServerStatusRunning,
				PID:          inst.pid(),
				GameServerID: inst.gameServerID,
			})
		}
	}
	if len(out) == 0 {
		return []protocol.ServerStatusPayload{{Status: protocol.ServerStatusStopped}}
	}
	return out
}

func (r *ProcessRunner) ensureInstancesLocked() {
	if r.instances == nil {
		r.instances = make(map[string]*managedInstance)
	}
}

func (r *ProcessRunner) getOrCreateInstanceLocked(key string) *managedInstance {
	r.ensureInstancesLocked()
	if inst := r.instances[key]; inst != nil {
		return inst
	}
	inst := &managedInstance{workDir: key}
	r.instances[key] = inst
	return inst
}

func (r *ProcessRunner) aliveInstanceLocked(key string) *managedInstance {
	if r.instances == nil {
		return nil
	}
	inst := r.instances[key]
	if inst.isAlive(r.DryRun) {
		return inst
	}
	return nil
}

func (inst *managedInstance) isAlive(dryRun bool) bool {
	if inst == nil {
		return false
	}
	if dryRun {
		return inst.dryPID != 0
	}
	return inst.cmd != nil && inst.cmd.Process != nil && processAlive(inst.cmd.Process.Pid)
}

func (inst *managedInstance) pid() int {
	if inst == nil {
		return 0
	}
	if inst.cmd != nil && inst.cmd.Process != nil {
		return inst.cmd.Process.Pid
	}
	return inst.dryPID
}

func (r *ProcessRunner) instanceByCmdLocked(cmd *exec.Cmd) *managedInstance {
	if cmd == nil {
		return nil
	}
	for _, inst := range r.instances {
		if inst != nil && inst.cmd == cmd {
			return inst
		}
	}
	return nil
}

func (r *ProcessRunner) clearInstanceLocked(inst *managedInstance) {
	if inst == nil {
		return
	}
	inst.closePipesLocked()
	inst.stopLogFollowLocked()
	inst.cmd = nil
	inst.stoppingGracefully = false
	inst.dryPID = 0
	inst.gameServerID = ""
	if r.instances != nil {
		delete(r.instances, inst.workDir)
	}
}

func (r *ProcessRunner) cleanupDeadProcessLocked() {
	if r.DryRun || r.instances == nil {
		return
	}
	for key, inst := range r.instances {
		if inst == nil {
			delete(r.instances, key)
			continue
		}
		if inst.cmd == nil || inst.cmd.Process == nil {
			continue
		}
		if processAlive(inst.cmd.Process.Pid) {
			continue
		}
		r.clearInstanceLocked(inst)
	}
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
	inst := r.instanceByCmdLocked(cmd)
	if inst == nil {
		r.mu.Unlock()
		return
	}
	stopping := inst.stoppingGracefully
	gameServerID := inst.gameServerID
	onStatus := r.onStatus
	if inst.cmd == cmd {
		r.clearInstanceLocked(inst)
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

func (r *ProcessRunner) adoptCmdForTest(cmd *exec.Cmd, workDir, gameServerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := instanceKey(workDir)
	inst := r.getOrCreateInstanceLocked(key)
	inst.cmd = cmd
	inst.gameServerID = gameServerID
	inst.workDir = key
}

func (r *ProcessRunner) hasManagedCmdLocked() bool {
	for _, inst := range r.instances {
		if inst != nil && inst.cmd != nil {
			return true
		}
	}
	return false
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
