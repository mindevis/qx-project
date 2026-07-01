//go:build windows

package browser

import "github.com/qxproject/qx/services/qxlauncher/internal/winutil"

func openURL(url string) error {
	return winutil.ShellExecuteOpen(url)
}
