package browser

import (
	"fmt"
	"path/filepath"
	"strings"
)

func OpenFolder(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return fmt.Errorf("empty path")
	}
	return openFolder(path)
}
