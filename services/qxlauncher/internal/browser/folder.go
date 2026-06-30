package browser

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

func OpenFolder(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	proc.HideConsole(cmd)
	return cmd.Start()
}
