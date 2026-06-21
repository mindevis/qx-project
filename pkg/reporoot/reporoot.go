package reporoot

import (
	"errors"
	"os"
	"path/filepath"
)

// Find walks up from startDir until go.work is found.
func Find(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.work not found")
		}
		dir = parent
	}
}

// ConfigFile returns path to name in the repo root when the file exists.
func ConfigFile(startDir, name string) string {
	root, err := Find(startDir)
	if err != nil {
		return ""
	}
	path := filepath.Join(root, name)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
