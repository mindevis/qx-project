package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

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

type assetJob struct {
	hash string
	url  string
	dest string
}

func (d *Downloader) EnsureAssets(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if manifest == nil || manifest.AssetIndex.ID == "" {
		return "", fmt.Errorf("missing asset index")
	}
	if manifest.InstanceID == "" {
		return "", fmt.Errorf("missing instance id")
	}
	assetsDir := d.InstanceAssetsDir(manifest.InstanceID)
	indexPath := filepath.Join(assetsDir, "indexes", manifest.AssetIndex.ID+".json")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return "", err
	}
	d.progressf("assets", "asset index %s …", manifest.AssetIndex.ID)
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
	if err := d.ensureIndexedAsset(ctx, assetsDir, cdn, index.Objects, languageAssetKey(DefaultGameLanguage)); err != nil {
		return "", err
	}

	jobs := make([]assetJob, 0)
	cached := 0
	for _, obj := range index.Objects {
		hash := strings.ToLower(strings.TrimSpace(obj.Hash))
		if len(hash) < 2 {
			continue
		}
		dest := filepath.Join(assetsDir, "objects", hash[:2], hash)
		if _, err := os.Stat(dest); err == nil {
			cached++
			continue
		}
		url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(cdn, "/"), hash[:2], hash)
		jobs = append(jobs, assetJob{hash: hash, url: url, dest: dest})
	}
	if err := d.downloadAssetsParallel(ctx, jobs, cached); err != nil {
		return "", err
	}
	return assetsDir, nil
}

func (d *Downloader) downloadAssetsParallel(ctx context.Context, jobs []assetJob, cached int) error {
	total := len(jobs)
	if total == 0 {
		d.progressf("assets", "objects ready (%d cached)", cached)
		return nil
	}
	d.progressf("assets", "downloading %d objects (%d cached) …", total, cached)

	const workers = 16
	jobsCh := make(chan assetJob)
	var wg sync.WaitGroup
	var done atomic.Int32
	var firstErr atomic.Pointer[error]

	worker := func() {
		defer wg.Done()
		for job := range jobsCh {
			if firstErr.Load() != nil {
				return
			}
			if err := d.downloadIfNeeded(ctx, job.url, job.dest, job.hash); err != nil {
				wrapped := fmt.Errorf("asset %s: %w", job.hash, err)
				firstErr.CompareAndSwap(nil, &wrapped)
				return
			}
			n := done.Add(1)
			if n == 1 || n%100 == 0 || int(n) == total {
				d.progressf("assets", "%d/%d objects", n, total)
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}
	for _, job := range jobs {
		if firstErr.Load() != nil {
			break
		}
		jobsCh <- job
	}
	close(jobsCh)
	wg.Wait()

	if errPtr := firstErr.Load(); errPtr != nil {
		return *errPtr
	}
	d.progressf("assets", "done (%d downloaded, %d cached)", total, cached)
	return nil
}

func (d *Downloader) ensureIndexedAsset(
	ctx context.Context,
	assetsDir, cdn string,
	objects map[string]assetObject,
	virtualPath string,
) error {
	obj, ok := objects[virtualPath]
	if !ok {
		return nil
	}
	hash := strings.ToLower(strings.TrimSpace(obj.Hash))
	if len(hash) < 2 {
		return nil
	}
	dest := filepath.Join(assetsDir, "objects", hash[:2], hash)
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(cdn, "/"), hash[:2], hash)
	if err := d.downloadIfNeeded(ctx, url, dest, hash); err != nil {
		return fmt.Errorf("asset %s: %w", virtualPath, err)
	}
	return nil
}
