package browser

import (
	"fmt"
	"runtime"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

func Open(url string) error {
	if url == "" {
		return fmt.Errorf("empty url")
	}
	switch runtime.GOOS {
	case "windows":
		return proc.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return proc.Command("open", url).Start()
	default:
		return proc.Command("xdg-open", url).Start()
	}
}
