package safepath

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// VettedAbsPath is an absolute path that passed validation (CodeQL path-injection barrier).
type VettedAbsPath string

// VettedAbs validates and returns an absolute path safe for os.* calls.
func VettedAbs(path string) (VettedAbsPath, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("missing path")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute")
	}
	clean := filepath.Clean(path)
	if clean != path {
		return "", fmt.Errorf("invalid path")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path")
	}
	return VettedAbsPath(clean), nil
}

func (p VettedAbsPath) String() string {
	return string(p)
}

// EnsureDir creates a validated server root directory.
func EnsureDir(dir string) error {
	safe, err := VettedAbs(dir)
	if err != nil {
		return err
	}
	return os.MkdirAll(safe.String(), 0o755)
}

// EnsureParent creates the parent directory of a validated file path.
func EnsureParent(filePath string) error {
	safe, err := VettedAbs(filePath)
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(safe.String()), 0o755)
}

// PartPath returns dest + ".part" when dest is already vetted.
func PartPath(dest string) (string, error) {
	safe, err := VettedAbs(dest)
	if err != nil {
		return "", err
	}
	return safe.String() + ".part", nil
}

// ReadFileBytes reads a file at a vetted absolute path.
func ReadFileBytes(path string) ([]byte, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(safe.String())
}

// WriteFileBytes writes a file at a vetted absolute path.
func WriteFileBytes(path string, data []byte, perm os.FileMode) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return os.WriteFile(safe.String(), data, perm)
}

// OpenRead opens a file at a vetted absolute path for reading.
func OpenRead(path string) (*os.File, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return os.Open(safe.String())
}

// Stat returns file metadata for a vetted absolute path.
func Stat(path string) (os.FileInfo, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return os.Stat(safe.String())
}

// ReadDir lists a vetted absolute directory.
func ReadDir(path string) ([]os.DirEntry, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(safe.String())
}

// Chmod sets permissions on a vetted absolute path.
func Chmod(path string, mode os.FileMode) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return os.Chmod(safe.String(), mode)
}

// Remove deletes a vetted absolute path.
func Remove(path string) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return os.Remove(safe.String())
}

// RemoveAll deletes a vetted absolute path and all of its children.
func RemoveAll(path string) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(safe.String())
}

// Rename moves a vetted absolute path to another vetted absolute path.
func Rename(oldPath, newPath string) error {
	oldSafe, err := VettedAbs(oldPath)
	if err != nil {
		return err
	}
	newSafe, err := VettedAbs(newPath)
	if err != nil {
		return err
	}
	return os.Rename(oldSafe.String(), newSafe.String())
}

// WriteStreamAtomic writes reader content to dest using a ".part" temp file.
func WriteStreamAtomic(dest string, r io.Reader) error {
	safeDest, err := VettedAbs(dest)
	if err != nil {
		return err
	}
	part, err := PartPath(safeDest.String())
	if err != nil {
		return err
	}
	partSafe, err := VettedAbs(part)
	if err != nil {
		return err
	}
	file, err := os.Create(partSafe.String())
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, r); err != nil {
		_ = file.Close()
		_ = Remove(partSafe.String())
		return err
	}
	if err := file.Close(); err != nil {
		_ = Remove(partSafe.String())
		return err
	}
	return Rename(partSafe.String(), safeDest.String())
}
