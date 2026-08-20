//go:build windows

package browser

import "github.com/qxproject/qx/services/qxlauncher/internal/winutil"

func openFolder(path string) error {
	return winutil.ShellExecuteOpen(path)
}
