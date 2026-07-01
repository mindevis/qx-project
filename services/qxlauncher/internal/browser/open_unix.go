//go:build !windows

package browser

import (
	"runtime"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

func openURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return proc.Command("open", url).Start()
	default:
		return proc.Command("xdg-open", url).Start()
	}
}
