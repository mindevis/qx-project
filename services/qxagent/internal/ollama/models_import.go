package ollama

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

var extraOllamaModelDirsFn = defaultExtraOllamaModelDirs

func (m *Manager) importExternalModels() {
	if m.DryRun {
		return
	}
	dest := m.ModelsDir()
	for _, src := range extraOllamaModelDirsFn(m.HomeDir(), dest) {
		_ = importOllamaModelStore(src, dest)
	}
}

func defaultExtraOllamaModelDirs(home, modelsDir string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(path string) {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == "." {
			return
		}
		if sameOrNestedPath(path, modelsDir) {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	add(filepath.Join(home, ".ollama", "models"))
	if userHome, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(userHome, ".ollama", "models"))
	}
	add("/root/.ollama/models")
	add("/usr/share/ollama/.ollama/models")
	add("/var/lib/ollama/.ollama/models")
	return out
}

func importOllamaModelStore(src, dest string) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)
	if src == "" || dest == "" || sameOrNestedPath(src, dest) {
		return nil
	}
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return nil
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info == nil || info.Mode()&os.ModeSymlink != 0 && info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if fileExists(target) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return linkOrCopyFile(path, target)
	})
}

func sameOrNestedPath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if a == b {
		return true
	}
	rel, err := filepath.Rel(b, a)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func linkOrCopyFile(src, dest string) error {
	if fileExists(dest) {
		return nil
	}
	if err := os.Symlink(src, dest); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
