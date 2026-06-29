package safepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRoot returns an absolute, cleaned directory path without ".." segments.
func ResolveRoot(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "", fmt.Errorf("missing path")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path")
	}
	return clean, nil
}

// Join joins fixed path elements under root. Each element must be a single name (no separators).
func Join(root string, elems ...string) (string, error) {
	root, err := ResolveRoot(root)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(elems)+1)
	parts = append(parts, root)
	for _, elem := range elems {
		elem = strings.TrimSpace(elem)
		if elem == "" {
			return "", fmt.Errorf("empty path element")
		}
		if elem == ".." || strings.ContainsRune(elem, os.PathSeparator) || strings.Contains(elem, "/") {
			return "", fmt.Errorf("invalid path element")
		}
		parts = append(parts, elem)
	}
	abs := filepath.Join(parts...)
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if err := underRoot(root, abs); err != nil {
		return "", err
	}
	return abs, nil
}

// JoinRel joins a relative path under root (blocks ".." traversal).
func JoinRel(root, rel string) (string, error) {
	root, err := ResolveRoot(root)
	if err != nil {
		return "", err
	}
	rel = strings.TrimSpace(rel)
	rel = strings.TrimPrefix(rel, "/")
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path")
	}
	abs := filepath.Join(root, rel)
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	if err := underRoot(root, abs); err != nil {
		return "", err
	}
	return abs, nil
}

// ResolveUnder resolves path relative to root when not absolute.
func ResolveUnder(root, path string) (string, error) {
	root, err := ResolveRoot(root)
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("missing path")
	}
	var abs string
	if filepath.IsAbs(path) {
		abs, err = filepath.Abs(path)
	} else {
		abs, err = filepath.Abs(filepath.Join(root, path))
	}
	if err != nil {
		return "", err
	}
	if err := underRoot(root, abs); err != nil {
		return "", err
	}
	return abs, nil
}

// ZipEntryBase returns the basename of a zip entry, rejecting traversal paths.
func ZipEntryBase(name string) (string, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if name == "" || strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("invalid zip entry")
	}
	base := filepath.Base(name)
	if base == "." || base == ".." {
		return "", fmt.Errorf("invalid zip entry")
	}
	return base, nil
}

func underRoot(root, abs string) error {
	if abs == root {
		return nil
	}
	if strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return nil
	}
	return fmt.Errorf("path outside root")
}
