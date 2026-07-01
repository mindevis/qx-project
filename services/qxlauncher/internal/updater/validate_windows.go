//go:build windows

package updater

import (
	"fmt"
	"os"
)

const minPEBytes = 64 * 1024 // reject tiny non-PE payloads before replacing the running binary

// validateWindowsExecutable checks that staging looks like a real PE image before install.
func validateWindowsExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() < minPEBytes {
		return fmt.Errorf("update payload too small (%d bytes)", info.Size())
	}
	header := make([]byte, 2)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	_, err = f.Read(header)
	_ = f.Close()
	if err != nil {
		return err
	}
	if header[0] != 'M' || header[1] != 'Z' {
		return fmt.Errorf("update payload is not a Windows executable")
	}
	return nil
}
