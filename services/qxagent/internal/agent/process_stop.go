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

	r.mu.Lock()
	if r.DryRun {
		slog.Info("dry-run stop server")
		r.dryPID = 0
		r.gameServerID = ""
		r.stopLogFollowLocked()
		r.mu.Unlock()
		return 0, nil
	}

	if strings.TrimSpace(workDir) != "" {
		if resolved, err := safepath.ResolveRoot(workDir); err == nil {
			workDir = resolved
		}
	} else {
		workDir = r.managedWorkDir
	}

	cmd := r.cmd
	stdin := r.stdin
	same := cmd != nil && cmd.Process != nil &&
		(workDir == "" || r.managedWorkDir == "" || workDirsMatch(r.managedWorkDir, workDir))
	if same {
		r.stoppingGracefully = true
	}
	r.mu.Unlock()

	pids := collectStopPIDs(cmd, same, workDir)
	if len(pids) == 0 {
		r.finishStop(cmd, same, workDir)
		return 0, nil
	}

	if graceful {
		if same && stdin != nil {
			_, _ = fmt.Fprintln(stdin, "stop")
			if waitUntilWorkDirIdle(pids, workDir, 2*time.Second) {
				r.finishStop(cmd, same, workDir)
				return 0, nil
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

	r.finishStop(cmd, same, workDir)
	return 0, nil
}

func (r *ProcessRunner) finishStop(cmd *exec.Cmd, same bool, workDir string) {
	if same {
		r.mu.Lock()
		if r.cmd == cmd {
			r.clearManagedProcessLocked()
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
