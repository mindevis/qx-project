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

// RemoveInstancesChild deletes the instance UUID folder.
// path may be the UUID directory itself or anything inside it.
// The removed directory is the direct child of a folder named "instances".
func RemoveInstancesChild(path string) error {
	abs, err := ResolveRoot(path)
	if err != nil {
		return err
	}
	target, err := instancesChildDir(abs)
	if err != nil {
		return err
	}
	return removeTree(target)
}

func instancesChildDir(abs string) (string, error) {
	current := abs
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("refusing to remove path outside instances")
		}
		if filepath.Base(parent) == "instances" {
			name := filepath.Base(current)
			if name == "" || name == "." || name == ".." || name == string(os.PathSeparator) {
				return "", fmt.Errorf("invalid instance dir")
			}
			return current, nil
		}
		current = parent
	}
}

func removeTree(path string) error {
	if _, err := Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	_ = chmodTreeWritable(path)
	if err := RemoveAll(path); err != nil && !os.IsNotExist(err) {
		_ = chmodTreeWritable(path)
		if err := RemoveAll(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if _, err := Stat(path); err == nil {
		return fmt.Errorf("instance directory still exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func chmodTreeWritable(path string) error {
	safe, err := VettedAbs(path)
	if err != nil {
		return err
	}
	return filepath.WalkDir(string(safe), func(child string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		vetted, vetErr := VettedAbs(child)
		if vetErr != nil {
			return nil
		}
		mode := os.FileMode(0o666)
		if d.IsDir() {
			mode = 0o777
		}
		_ = vetted.chmod(mode)
		return nil
	})
}

// CopyInstancesChild copies one instances/{id} tree onto another UUID folder.
func CopyInstancesChild(src, dest string) error {
	srcAbs, err := ResolveRoot(src)
	if err != nil {
		return err
	}
	destAbs, err := ResolveRoot(dest)
	if err != nil {
		return err
	}
	srcRoot, err := instancesChildDir(srcAbs)
	if err != nil {
		return err
	}
	destRoot, err := instancesChildDir(destAbs)
	if err != nil {
		return err
	}
	if srcRoot == destRoot {
		return fmt.Errorf("source and destination are the same")
	}
	sep := string(os.PathSeparator)
	if strings.HasPrefix(destRoot, srcRoot+sep) {
		return fmt.Errorf("destination is inside source")
	}
	if _, err := Stat(srcRoot); err != nil {
		return err
	}
	return copyTree(srcRoot, destRoot)
}

func copyTree(src, dest string) error {
	if err := EnsureDir(dest); err != nil {
		return err
	}
	entries, err := ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if skipCopyName(entry.Name()) {
			continue
		}
		childSrc, err := Join(src, entry.Name())
		if err != nil {
			return err
		}
		childDest, err := Join(dest, entry.Name())
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if err := copyTree(childSrc, childDest); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(childSrc, childDest); err != nil {
			return err
		}
	}
	return nil
}

func skipCopyName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "session.lock", "client.lock":
		return true
	default:
		return false
	}
}

func copyFile(src, dest string) error {
	in, err := OpenRead(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := EnsureParent(dest); err != nil {
		return err
	}
	return WriteStreamAtomic(dest, in)
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
