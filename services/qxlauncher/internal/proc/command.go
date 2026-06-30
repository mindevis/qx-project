package proc

import (
	"context"
	"os/exec"
)

// Command is like exec.Command with HideConsole applied.
func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	HideConsole(cmd)
	return cmd
}

// CommandContext is like exec.CommandContext with HideConsole applied.
func CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	HideConsole(cmd)
	return cmd
}
