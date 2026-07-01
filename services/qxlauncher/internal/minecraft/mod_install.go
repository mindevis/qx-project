package minecraft

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// InstallResource downloads a catalog file into the instance resource folder.
func (d *Downloader) InstallResource(ctx context.Context, instanceID, folder, filename, url string) error {
	if instanceID == "" || folder == "" || filename == "" || url == "" {
		return fmt.Errorf("install resource: missing required fields")
	}
	destDir := filepath.Join(d.InstanceGameDir(instanceID), folder)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create resource dir: %w", err)
	}
	dest := filepath.Join(destDir, filepath.Base(filename))
	return d.downloadIfNeeded(ctx, url, dest, "")
}
