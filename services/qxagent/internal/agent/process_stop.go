package agent

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/safepath"
)

const agentPidFileName = ".qxagent.pid"

func (r *ProcessRunner) StopIfWorkDir(workDir string, graceful bool, timeout time.Duration) {
	_, _ = r.StopTarget(graceful, timeout, workDir)
}

func (r *ProcessRunner) Stop(graceful bool, timeout time.Duration) (int, error) {
	return r.StopTarget(graceful, timeout, "")
}

func (r *ProcessRunner) StopTarget(graceful bool, timeout time.Duration, workDir string) (int, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	workDir = strings.TrimSpace(workDir)
	if workDir != "" {
		if resolved, err := safepath.ResolveRoot(workDir); err == nil {
			workDir = resolved
		}
	}

	r.mu.Lock()
	if r.DryRun {
		r.stopDryRunLocked(workDir)
		r.mu.Unlock()
		return 0, nil
	}
	targets := r.instancesToStopLocked(workDir)
	r.mu.Unlock()

	if len(targets) == 0 {
		r.stopWorkDirProcesses(workDir, graceful, timeout)
		return 0, nil
	}
	for _, inst := range targets {
		r.stopInstance(inst, graceful, timeout)
	}
	return 0, nil
}

func (r *ProcessRunner) stopDryRunLocked(workDir string) {
	slog.Info("dry-run stop server", "work_dir", workDir)
	if workDir == "" {
		for _, inst := range r.instances {
			if inst != nil {
				inst.stopLogFollowLocked()
				inst.dryPID = 0
				inst.gameServerID = ""
			}
		}
		r.instances = nil
		return
	}
	if inst := r.instances[workDir]; inst != nil {
		inst.stopLogFollowLocked()
		inst.dryPID = 0
		inst.gameServerID = ""
		delete(r.instances, workDir)
	}
}

func (r *ProcessRunner) instancesToStopLocked(workDir string) []*managedInstance {
	r.ensureInstancesLocked()
	if workDir != "" {
		if inst := r.instances[workDir]; inst != nil {
			return []*managedInstance{inst}
		}
		for key, inst := range r.instances {
			if inst != nil && workDirsMatch(key, workDir) {
				return []*managedInstance{inst}
			}
		}
		return nil
	}
	out := make([]*managedInstance, 0, len(r.instances))
	for _, inst := range r.instances {
		if inst != nil {
			out = append(out, inst)
		}
	}
	return out
}

func (r *ProcessRunner) stopInstance(inst *managedInstance, graceful bool, timeout time.Duration) {
	if inst == nil {
		return
	}
	r.mu.Lock()
	cmd := inst.cmd
	stdin := inst.stdin
	workDir := inst.workDir
	same := cmd != nil && cmd.Process != nil
	if same {
		inst.stoppingGracefully = true
	}
	r.mu.Unlock()

	pids := collectStopPIDs(cmd, same, workDir)
	if len(pids) == 0 {
		r.finishStop(inst, cmd, same, workDir)
		return
	}

	if graceful {
		if same && stdin != nil {
			_, _ = fmt.Fprintln(stdin, "stop")
			if waitUntilWorkDirIdle(pids, workDir, 2*time.Second) {
				r.finishStop(inst, cmd, same, workDir)
				return
			}
		}
		for _, pid := range collectStopPIDs(cmd, same, workDir) {
			signalProcessTree(pid, false)
		}
		remain := timeout
		if remain < time.Second {
			remain = time.Second
		}
		_ = waitUntilWorkDirIdle(pids, workDir, remain)
	}

	pids = collectStopPIDs(cmd, same, workDir)
	for _, pid := range pids {
		signalProcessTree(pid, true)
	}
	_ = waitUntilWorkDirIdle(pids, workDir, 3*time.Second)

	r.finishStop(inst, cmd, same, workDir)
}

func (r *ProcessRunner) stopWorkDirProcesses(workDir string, graceful bool, timeout time.Duration) {
	pids := collectStopPIDs(nil, false, workDir)
	if len(pids) == 0 {
		removePidFile(workDir)
		return
	}
	if graceful {
		for _, pid := range collectStopPIDs(nil, false, workDir) {
			signalProcessTree(pid, false)
		}
		remain := timeout
		if remain < time.Second {
			remain = time.Second
		}
		_ = waitUntilWorkDirIdle(pids, workDir, remain)
	}
	pids = collectStopPIDs(nil, false, workDir)
	for _, pid := range pids {
		signalProcessTree(pid, true)
	}
	_ = waitUntilWorkDirIdle(pids, workDir, 3*time.Second)
	removePidFile(workDir)
}

func (r *ProcessRunner) finishStop(inst *managedInstance, cmd *exec.Cmd, same bool, workDir string) {
	if same && inst != nil {
		r.mu.Lock()
		if inst.cmd == cmd {
			r.clearInstanceLocked(inst)
		}
		r.mu.Unlock()
	}
	removePidFile(workDir)
}

func collectStopPIDs(cmd *exec.Cmd, includeCmd bool, workDir string) []int {
	var pids []int
	if includeCmd && cmd != nil && cmd.Process != nil {
		pids = append(pids, cmd.Process.Pid)
	}
	if pid := readPidFile(workDir); pid > 0 {
		pids = append(pids, pid)
	}
	pids = append(pids, pidsInWorkDir(workDir)...)
	return uniquePositivePIDs(pids)
}

func waitUntilWorkDirIdle(initial []int, workDir string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !anyStopTargetRunning(initial, workDir) {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func anyStopTargetRunning(initial []int, workDir string) bool {
	for _, pid := range uniquePositivePIDs(append(append([]int{}, initial...), pidsInWorkDir(workDir)...)) {
		if processRunning(pid) {
			return true
		}
	}
	if pid := readPidFile(workDir); pid > 0 && processRunning(pid) {
		return true
	}
	return false
}

func uniquePositivePIDs(pids []int) []int {
	seen := make(map[int]struct{}, len(pids))
	self := os.Getpid()
	out := make([]int, 0, len(pids))
	for _, pid := range pids {
		if pid <= 1 || pid == self {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		out = append(out, pid)
	}
	return out
}

func pidFilePath(workDir string) string {
	path, err := safepath.Join(workDir, agentPidFileName)
	if err != nil {
		return ""
	}
	return path
}

func writePidFile(workDir string, pid int) {
	path := pidFilePath(workDir)
	if path == "" || pid <= 0 {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func readPidFile(workDir string) int {
	path := pidFilePath(workDir)
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return 0
	}
	return pid
}

func removePidFile(workDir string) {
	path := pidFilePath(workDir)
	if path == "" {
		return
	}
	_ = os.Remove(path)
}
