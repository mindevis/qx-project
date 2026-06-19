package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

const defaultAssetsCDN = "https://resources.download.minecraft.net"

type assetIndexFile struct {
	Objects map[string]assetObject `json:"objects"`
}

type assetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

func (d *Downloader) AssetsDir() string {
	return filepath.Join(d.RootDir, "assets")
}

func (d *Downloader) EnsureAssets(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if manifest == nil || manifest.AssetIndex.ID == "" {
		return "", fmt.Errorf("missing asset index")
	}
	assetsDir := d.AssetsDir()
	indexPath := filepath.Join(assetsDir, "indexes", manifest.AssetIndex.ID+".json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return "", err
	}
	if manifest.AssetIndex.URL != "" {
		if err := d.downloadIfNeeded(ctx, manifest.AssetIndex.URL, indexPath, manifest.AssetIndex.Sha1); err != nil {
			return "", fmt.Errorf("asset index: %w", err)
		}
	} else if _, err := os.Stat(indexPath); err != nil {
		return "", fmt.Errorf("asset index file missing")
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", err
	}
	var index assetIndexFile
	if err := json.Unmarshal(data, &index); err != nil {
		return "", fmt.Errorf("parse asset index: %w", err)
	}

	cdn := defaultAssetsCDN
	if d.AssetsCDN != "" {
		cdn = strings.TrimRight(d.AssetsCDN, "/")
	}
	for _, obj := range index.Objects {
		hash := strings.ToLower(strings.TrimSpace(obj.Hash))
		if len(hash) < 2 {
			continue
		}
		dest := filepath.Join(assetsDir, "objects", hash[:2], hash)
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(cdn, "/"), hash[:2], hash)
		if err := d.downloadIfNeeded(ctx, url, dest, hash); err != nil {
			return "", fmt.Errorf("asset %s: %w", hash, err)
		}
	}
	return assetsDir, nil
}
