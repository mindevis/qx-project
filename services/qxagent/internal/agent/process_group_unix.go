//go:build !windows

package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/qxproject/qx/pkg/safepath"
)

func configureProcessGroup(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalProcessTree(pid int, force bool) {
	if pid <= 1 {
		return
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	if err := syscall.Kill(-pid, sig); err != nil {
		_ = syscall.Kill(pid, sig)
	}
}

func pidsInWorkDir(workDir string) []int {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return nil
	}
	resolved, err := safepath.ResolveRoot(workDir)
	if err != nil {
		return nil
	}
	if resolved == string(os.PathSeparator) {
		return nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()
	prefix := resolved + string(os.PathSeparator)
	var pids []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 1 || pid == self {
			continue
		}
		cwd, err := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
		if err != nil {
			continue
		}
		cwd = filepath.Clean(cwd)
		if cwd == resolved || strings.HasPrefix(cwd, prefix) {
			pids = append(pids, pid)
		}
	}
	return pids
}
