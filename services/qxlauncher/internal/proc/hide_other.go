//go:build !windows

package proc

import "os/exec"

func HideConsole(cmd *exec.Cmd) {
	_ = cmd
}
