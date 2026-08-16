package minecraft

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/qxproject/qx/pkg/safepath"
)

// InstallResource downloads a catalog file into the instance resource folder.
func (d *Downloader) InstallResource(ctx context.Context, instanceID, folder, filename, url string) error {
	if instanceID == "" || folder == "" || filename == "" || url == "" {
		return fmt.Errorf("install resource: missing required fields")
	}
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return err
	}
	rel := filepath.ToSlash(filepath.Join(folder, filepath.Base(filename)))
	dest, err := safepath.JoinRel(gameDir, rel)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(dest); err != nil {
		return fmt.Errorf("create resource dir: %w", err)
	}
	return d.downloadIfNeeded(ctx, url, dest, "")
}
