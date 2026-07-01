package fs

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

const contentDownloadTimeout = 5 * time.Minute

func ListPlugins(workDir string) ([]protocol.FileEntry, error) {
	return ListDir(workDir, "plugins")
}

func ListDatapacks(workDir string) ([]protocol.FileEntry, error) {
	world, err := WorldFolder(workDir)
	if err != nil {
		return nil, err
	}
	return ListDir(workDir, filepath.Join(world, "datapacks"))
}

func WorldFolder(workDir string) (string, error) {
	levelName := "world"
	props, err := ReadServerProperties(workDir)
	if err != nil {
		return "", err
	}
	for _, entry := range props {
		if entry.Key == "level-name" && strings.TrimSpace(entry.Value) != "" {
			levelName = strings.TrimSpace(entry.Value)
			break
		}
	}
	if strings.Contains(levelName, "..") || strings.ContainsAny(levelName, `/\`) {
		return "", fmt.Errorf("invalid level-name")
	}
	return levelName, nil
}

func ContentRelPath(workDir, serverType, contentKind, filename string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return "", fmt.Errorf("invalid filename")
	}
	switch strings.ToLower(strings.TrimSpace(contentKind)) {
	case "mod":
		folder := modsFolderFor(serverType)
		if folder == "" {
			return "", fmt.Errorf("server type does not support mods")
		}
		return filepath.ToSlash(filepath.Join(folder, filename)), nil
	case "plugin":
		return filepath.ToSlash(filepath.Join("plugins", filename)), nil
	case "datapack":
		world, err := WorldFolder(workDir)
		if err != nil {
			return "", err
		}
		return filepath.ToSlash(filepath.Join(world, "datapacks", filename)), nil
	default:
		return "", fmt.Errorf("unknown content kind %q", contentKind)
	}
}

func InstallContentFile(ctx context.Context, workDir, relPath, downloadURL string) error {
	relPath = strings.TrimSpace(relPath)
	downloadURL = strings.TrimSpace(downloadURL)
	if relPath == "" || downloadURL == "" {
		return fmt.Errorf("rel_path and download_url required")
	}
	abs, err := safepath.JoinRel(workDir, relPath)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	downloadCtx, cancel := context.WithTimeout(ctx, contentDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 0}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: http %d", downloadURL, res.StatusCode)
	}
	return safepath.WriteStreamAtomic(abs, res.Body)
}
