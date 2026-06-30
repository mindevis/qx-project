package browser

import (
	"fmt"
	"runtime"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

func OpenFolder(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	switch runtime.GOOS {
	case "windows":
		return proc.Command("explorer", path).Start()
	case "darwin":
		return proc.Command("open", path).Start()
	default:
		return proc.Command("xdg-open", path).Start()
	}
}
