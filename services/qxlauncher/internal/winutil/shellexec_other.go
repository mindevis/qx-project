//go:build !windows

package winutil

import "fmt"

func ShellExecuteOpen(string) error {
	return fmt.Errorf("ShellExecuteOpen is only supported on Windows")
}
