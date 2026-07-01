//go:build windows

package updater

import (
	"os"
	"path/filepath"
)

// CleanupPreviousBackup removes the backup left by a prior in-place update.
// The previous process exits before the new one starts, so the file is not locked.
func CleanupPreviousBackup() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return
	}
	_ = os.Remove(backupPath(exe))
}
