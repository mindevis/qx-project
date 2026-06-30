//go:build windows

package proc

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func HideConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow,
		HideWindow:    true,
	}
}
