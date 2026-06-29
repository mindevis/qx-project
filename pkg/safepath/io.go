package safepath

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func assertVettedAbs(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("missing path")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute")
	}
	clean := filepath.Clean(path)
	if clean != path {
		return fmt.Errorf("invalid path")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("invalid path")
	}
	return nil
}

// EnsureDir creates a validated server root directory.
func EnsureDir(dir string) error {
	if err := assertVettedAbs(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// EnsureParent creates the parent directory of a validated file path.
func EnsureParent(filePath string) error {
	if err := assertVettedAbs(filePath); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(filePath), 0o755)
}

// PartPath returns dest + ".part" when dest is already vetted.
func PartPath(dest string) (string, error) {
	if err := assertVettedAbs(dest); err != nil {
		return "", err
	}
	return dest + ".part", nil
}

// ReadFileBytes reads a file at a vetted absolute path.
func ReadFileBytes(path string) ([]byte, error) {
	if err := assertVettedAbs(path); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// WriteFileBytes writes a file at a vetted absolute path.
func WriteFileBytes(path string, data []byte, perm os.FileMode) error {
	if err := assertVettedAbs(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, perm)
}

// OpenRead opens a file at a vetted absolute path for reading.
func OpenRead(path string) (*os.File, error) {
	if err := assertVettedAbs(path); err != nil {
		return nil, err
	}
	return os.Open(path)
}

// Stat returns file metadata for a vetted absolute path.
func Stat(path string) (os.FileInfo, error) {
	if err := assertVettedAbs(path); err != nil {
		return nil, err
	}
	return os.Stat(path)
}

// ReadDir lists a vetted absolute directory.
func ReadDir(path string) ([]os.DirEntry, error) {
	if err := assertVettedAbs(path); err != nil {
		return nil, err
	}
	return os.ReadDir(path)
}

// Chmod sets permissions on a vetted absolute path.
func Chmod(path string, mode os.FileMode) error {
	if err := assertVettedAbs(path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// Remove deletes a vetted absolute path.
func Remove(path string) error {
	if err := assertVettedAbs(path); err != nil {
		return err
	}
	return os.Remove(path)
}

// Rename moves a vetted absolute path to another vetted absolute path.
func Rename(oldPath, newPath string) error {
	if err := assertVettedAbs(oldPath); err != nil {
		return err
	}
	if err := assertVettedAbs(newPath); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

// WriteStreamAtomic writes reader content to dest using a ".part" temp file.
func WriteStreamAtomic(dest string, r io.Reader) error {
	part, err := PartPath(dest)
	if err != nil {
		return err
	}
	file, err := os.Create(part)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, r); err != nil {
		_ = file.Close()
		_ = Remove(part)
		return err
	}
	if err := file.Close(); err != nil {
		_ = Remove(part)
		return err
	}
	return Rename(part, dest)
}
