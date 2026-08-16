package mojangjava

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/pkg/safepath"
)

// RuntimeCatalogURL is Mojang's official Java runtime index.
var RuntimeCatalogURL = "https://piston-meta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json"

type runtimeCatalog map[string]map[string][]runtimeEntry

type runtimeEntry struct {
	Manifest struct {
		URL  string `json:"url"`
		Sha1 string `json:"sha1"`
	} `json:"manifest"`
	Version struct {
		Name string `json:"name"`
	} `json:"version"`
}

type packageManifest struct {
	Files map[string]packageFile `json:"files"`
}

type packageFile struct {
	Type       string                            `json:"type"`
	Downloads  map[string]mcmanifest.DownloadFile `json:"downloads,omitempty"`
	Executable bool                              `json:"executable,omitempty"`
}

func (m *Manager) resolveRuntimeEntry(ctx context.Context, platform, component string) (*runtimeEntry, error) {
	catalogURL := RuntimeCatalogURL
	if m.CatalogURL != "" {
		catalogURL = m.CatalogURL
	}
	body, err := m.fetchBytes(ctx, catalogURL)
	if err != nil {
		return nil, fmt.Errorf("java runtime catalog: %w", err)
	}
	var catalog runtimeCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("parse java runtime catalog: %w", err)
	}
	byComponent, ok := catalog[platform]
	if !ok {
		return nil, fmt.Errorf("platform %q not in java catalog", platform)
	}
	entries, ok := byComponent[component]
	if !ok || len(entries) == 0 {
		return nil, fmt.Errorf("component %q not available for %q", component, platform)
	}
	entry := entries[0]
	if entry.Manifest.URL == "" {
		return nil, fmt.Errorf("missing package manifest url for %s/%s", platform, component)
	}
	return &entry, nil
}

func (m *Manager) installPackage(ctx context.Context, manifestURL, destDir string) error {
	destDir, err := safepath.ResolveRoot(destDir)
	if err != nil {
		return err
	}
	body, err := m.fetchBytes(ctx, manifestURL)
	if err != nil {
		return fmt.Errorf("java package manifest: %w", err)
	}
	var pkg packageManifest
	if err := json.Unmarshal(body, &pkg); err != nil {
		return fmt.Errorf("parse java package manifest: %w", err)
	}
	for rel, file := range pkg.Files {
		if file.Type != "file" {
			continue
		}
		dl := pickDownload(file.Downloads)
		if dl.URL == "" {
			continue
		}
		dest, err := safepath.JoinRel(destDir, rel)
		if err != nil {
			return fmt.Errorf("java file %s: %w", rel, err)
		}
		if err := m.downloadIfNeeded(ctx, dl.URL, dest, dl.Sha1); err != nil {
			return fmt.Errorf("java file %s: %w", rel, err)
		}
		if file.Executable {
			_ = safepath.Chmod(dest, 0o755)
		}
	}
	return nil
}

func pickDownload(downloads map[string]mcmanifest.DownloadFile) mcmanifest.DownloadFile {
	if dl, ok := downloads["raw"]; ok && dl.URL != "" {
		return dl
	}
	for _, dl := range downloads {
		if dl.URL != "" {
			return dl
		}
	}
	return mcmanifest.DownloadFile{}
}

func (m *Manager) fetchBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := m.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("http %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(res.Body)
}

func (m *Manager) downloadIfNeeded(ctx context.Context, url, dest, sha1hex string) error {
	if sha1hex != "" {
		if b, err := safepath.ReadFileBytes(dest); err == nil {
			if hex.EncodeToString(sha1Sum(b)) == strings.ToLower(sha1hex) {
				return nil
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := m.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", url, res.StatusCode)
	}
	dest, err = safepath.ResolveRoot(dest)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(dest); err != nil {
		return err
	}
	return safepath.WriteStreamAtomic(dest, res.Body)
}

func sha1Sum(b []byte) []byte {
	sum := sha1.Sum(b)
	return sum[:]
}
