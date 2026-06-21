package envfile

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/reporoot"
)

// Load reads KEY=VALUE lines into the process environment.
// Existing environment variables are not overwritten.
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

// RepoRoot walks up from startDir until go.work is found.
func RepoRoot(startDir string) (string, error) {
	return reporoot.Find(startDir)
}

// LoadRepoDotEnv loads .env from the repository root (directory containing go.work).
func LoadRepoDotEnv(startDir string) error {
	root, err := RepoRoot(startDir)
	if err != nil {
		return err
	}
	path := filepath.Join(root, ".env")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return Load(path)
}
