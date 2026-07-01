//go:build windows

package agent

import (
	"syscall"
)

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	const (
		queryLimited = 0x1000
		stillActive  = 259
	)
	handle, err := syscall.OpenProcess(queryLimited, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)
	var code uint32
	if err := syscall.GetExitCodeProcess(handle, &code); err != nil {
		return false
	}
	return code == stillActive
}
