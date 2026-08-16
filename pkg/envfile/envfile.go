package envfile

import (
	"bufio"
	"os"
	"strings"

	"github.com/qxproject/qx/pkg/reporoot"
	"github.com/qxproject/qx/pkg/safepath"
)

// Load reads KEY=VALUE lines into the process environment.
// Existing environment variables are not overwritten.
func Load(path string) error {
	abs, err := safepath.ResolveRoot(path)
	if err != nil {
		return err
	}
	f, err := safepath.OpenRead(abs)
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
	path, err := safepath.Join(root, ".env")
	if err != nil {
		return err
	}
	if _, err := safepath.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return Load(path)
}
