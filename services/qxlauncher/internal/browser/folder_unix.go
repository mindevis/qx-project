//go:build !windows

package browser

import (
	"runtime"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

func openFolder(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return proc.Command("open", path).Start()
	default:
		return proc.Command("xdg-open", path).Start()
	}
}
