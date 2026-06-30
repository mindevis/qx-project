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

// vettedPath returns path after vetting (CodeQL path-injection barrier).
func vettedPath(path string) (string, error) {
	if err := assertVettedAbs(path); err != nil {
		return "", err
	}
	return path, nil
}

// EnsureDir creates a validated server root directory.
func EnsureDir(dir string) error {
	safe, err := vettedPath(dir)
	if err != nil {
		return err
	}
	return os.MkdirAll(safe, 0o755)
}

// EnsureParent creates the parent directory of a validated file path.
func EnsureParent(filePath string) error {
	safe, err := vettedPath(filePath)
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(safe), 0o755)
}

// PartPath returns dest + ".part" when dest is already vetted.
func PartPath(dest string) (string, error) {
	safe, err := vettedPath(dest)
	if err != nil {
		return "", err
	}
	return safe + ".part", nil
}

// ReadFileBytes reads a file at a vetted absolute path.
func ReadFileBytes(path string) ([]byte, error) {
	safe, err := vettedPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(safe)
}

// WriteFileBytes writes a file at a vetted absolute path.
func WriteFileBytes(path string, data []byte, perm os.FileMode) error {
	safe, err := vettedPath(path)
	if err != nil {
		return err
	}
	return os.WriteFile(safe, data, perm)
}

// OpenRead opens a file at a vetted absolute path for reading.
func OpenRead(path string) (*os.File, error) {
	safe, err := vettedPath(path)
	if err != nil {
		return nil, err
	}
	return os.Open(safe)
}

// Stat returns file metadata for a vetted absolute path.
func Stat(path string) (os.FileInfo, error) {
	safe, err := vettedPath(path)
	if err != nil {
		return nil, err
	}
	return os.Stat(safe)
}

// ReadDir lists a vetted absolute directory.
func ReadDir(path string) ([]os.DirEntry, error) {
	safe, err := vettedPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(safe)
}

// Chmod sets permissions on a vetted absolute path.
func Chmod(path string, mode os.FileMode) error {
	safe, err := vettedPath(path)
	if err != nil {
		return err
	}
	return os.Chmod(safe, mode)
}

// Remove deletes a vetted absolute path.
func Remove(path string) error {
	safe, err := vettedPath(path)
	if err != nil {
		return err
	}
	return os.Remove(safe)
}

// Rename moves a vetted absolute path to another vetted absolute path.
func Rename(oldPath, newPath string) error {
	oldSafe, err := vettedPath(oldPath)
	if err != nil {
		return err
	}
	newSafe, err := vettedPath(newPath)
	if err != nil {
		return err
	}
	return os.Rename(oldSafe, newSafe)
}

// WriteStreamAtomic writes reader content to dest using a ".part" temp file.
func WriteStreamAtomic(dest string, r io.Reader) error {
	safeDest, err := vettedPath(dest)
	if err != nil {
		return err
	}
	part, err := PartPath(safeDest)
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
	return Rename(part, safeDest)
}
