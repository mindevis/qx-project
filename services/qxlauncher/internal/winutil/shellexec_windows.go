//go:build windows

package winutil

import (
	"fmt"
	"syscall"
	"unsafe"
)

const swShowNormal = 1

var (
	shell32        = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

// ShellExecuteOpen opens a document or URL with the default handler.
func ShellExecuteOpen(target string) error {
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	openPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(openPtr)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		swShowNormal,
	)
	if ret <= 32 {
		if callErr != syscall.Errno(0) {
			return callErr
		}
		return fmt.Errorf("ShellExecute failed (code %d)", ret)
	}
	return nil
}
