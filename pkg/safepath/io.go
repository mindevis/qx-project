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

// VettedAbs validates and rebuilds an absolute path safe for os.* calls.
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
	vol := filepath.VolumeName(clean)
	rest := clean
	if vol != "" {
		rest = strings.TrimPrefix(clean, vol)
	}
	var parts []string
	for _, part := range strings.Split(rest, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		if part == ".." || strings.ContainsAny(part, `/\`) {
			return "", fmt.Errorf("invalid path")
		}
		parts = append(parts, part)
	}
	rebuilt := string(os.PathSeparator) + strings.Join(parts, string(os.PathSeparator))
	if vol != "" {
		rebuilt = vol + string(os.PathSeparator)
		if len(parts) > 0 {
			rebuilt = vol + string(os.PathSeparator) + strings.Join(parts, string(os.PathSeparator))
		}
	}
	if filepath.Clean(rebuilt) != clean {
		return "", fmt.Errorf("invalid path")
	}
	return VettedAbsPath(rebuilt), nil
}

func (p VettedAbsPath) String() string {
	return string(p)
}

func (p VettedAbsPath) mkdirAll(perm os.FileMode) error {
	return os.MkdirAll(string(p), perm)
}

func (p VettedAbsPath) create() (*os.File, error) {
	return os.Create(string(p))
}

func (p VettedAbsPath) open() (*os.File, error) {
	return os.Open(string(p))
}

func (p VettedAbsPath) openFile(flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(string(p), flag, perm)
}

func (p VettedAbsPath) readFile() ([]byte, error) {
	return os.ReadFile(string(p))
}

func (p VettedAbsPath) writeFile(data []byte, perm os.FileMode) error {
	return os.WriteFile(string(p), data, perm)
}

func (p VettedAbsPath) stat() (os.FileInfo, error) {
	return os.Stat(string(p))
}

func (p VettedAbsPath) readDir() ([]os.DirEntry, error) {
	return os.ReadDir(string(p))
}

func (p VettedAbsPath) chmod(mode os.FileMode) error {
	return os.Chmod(string(p), mode)
}

func (p VettedAbsPath) remove() error {
	return os.Remove(string(p))
}

func (p VettedAbsPath) removeAll() error {
	return os.RemoveAll(string(p))
}

func (p VettedAbsPath) rename(dest VettedAbsPath) error {
	return os.Rename(string(p), string(dest))
}

func (p VettedAbsPath) parent() (VettedAbsPath, error) {
	return VettedAbs(filepath.Dir(string(p)))
}

// EnsureDir creates a validated server root directory.
func EnsureDir(dir string) error {
	safe, err := VettedAbs(dir)
	if err != nil {
		return err
	}
	return safe.mkdirAll(0o755)
}

// EnsureParent creates the parent directory of a validated file path.
func EnsureParent(filePath string) error {
	safe, err := VettedAbs(filePath)
	if err != nil {
		return err
	}
	parent, err := safe.parent()
	if err != nil {
		return err
	}
	return parent.mkdirAll(0o755)
}

// PartPath returns dest + ".part" after both paths pass validation.
func PartPath(dest string) (VettedAbsPath, error) {
	safe, err := VettedAbs(dest)
	if err != nil {
		return "", err
	}
	return VettedAbs(string(safe) + ".part")
}

// ReadFileBytes reads a file at a vetted absolute path.
func ReadFileBytes(path string) ([]byte, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return safe.readFile()
}

// WriteFileBytes writes a file at a vetted absolute path.
func WriteFileBytes(path string, data []byte, perm os.FileMode) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return safe.writeFile(data, perm)
}

// Create creates a file at a vetted absolute path.
func Create(path string) (*os.File, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return safe.create()
}

// OpenRead opens a file at a vetted absolute path for reading.
func OpenRead(path string) (*os.File, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return safe.open()
}

// OpenFile opens a file at a vetted absolute path.
func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return safe.openFile(flag, perm)
}

// Stat returns file metadata for a vetted absolute path.
func Stat(path string) (os.FileInfo, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return safe.stat()
}

// ReadDir lists a vetted absolute directory.
func ReadDir(path string) ([]os.DirEntry, error) {
	safe, err := VettedAbs(path)
	if err != nil {
		return nil, err
	}
	return safe.readDir()
}

// Chmod sets permissions on a vetted absolute path.
func Chmod(path string, mode os.FileMode) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return safe.chmod(mode)
}

// Remove deletes a vetted absolute path.
func Remove(path string) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return safe.remove()
}

// RemoveAll deletes a vetted absolute path and all of its children.
func RemoveAll(path string) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return safe.removeAll()
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
	return oldSafe.rename(newSafe)
}

// WriteStreamAtomic writes reader content to dest using a ".part" temp file.
func WriteStreamAtomic(dest string, r io.Reader) error {
	safeDest, err := VettedAbs(dest)
	if err != nil {
		return err
	}
	part, err := PartPath(string(safeDest))
	if err != nil {
		return err
	}
	file, err := part.create()
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, r); err != nil {
		_ = file.Close()
		_ = part.remove()
		return err
	}
	if err := file.Close(); err != nil {
		_ = part.remove()
		return err
	}
	return part.rename(safeDest)
}
