package minecraft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

var javaRuntimeCatalogURL = "https://piston-meta.mojang.com/v1/products/java-runtime/2ec0cc96c44e5a76b9c8b7c39df7210883d12871/all.json"

type javaRuntimeCatalog map[string]map[string][]javaRuntimeEntry

type javaRuntimeEntry struct {
	Manifest struct {
		URL  string `json:"url"`
		Sha1 string `json:"sha1"`
	} `json:"manifest"`
	Version struct {
		Name string `json:"name"`
	} `json:"version"`
}

type javaPackageManifest struct {
	Files map[string]javaPackageFile `json:"files"`
}

type javaPackageFile struct {
	Type        string                       `json:"type"`
	Downloads   map[string]mcmanifest.DownloadFile `json:"downloads,omitempty"`
	Executable  bool                         `json:"executable,omitempty"`
}

// EnsureJava returns a Mojang-provided Java binary for the launch manifest.
// QX_JAVA skips download; QX_SKIP_JAVA_DOWNLOAD uses system Java (tests).
func (d *Downloader) EnsureJava(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if custom := os.Getenv("QX_JAVA"); custom != "" {
		return custom, nil
	}
	if os.Getenv("QX_SKIP_JAVA_DOWNLOAD") == "1" {
		return ResolveJavaBin(), nil
	}
	if manifest == nil {
		return "", fmt.Errorf("missing manifest")
	}

	component := strings.TrimSpace(manifest.JavaComponent)
	if component == "" {
		component = componentForJavaMajor(manifest.JavaMajor)
	}
	if component == "" {
		return "", fmt.Errorf("unknown java component for major %d", manifest.JavaMajor)
	}

	platform := javaPlatformKey()
	entry, err := d.resolveJavaRuntimeEntry(ctx, platform, component)
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(d.RootDir, "java", component, entry.Version.Name)
	if bin, ok := cachedJavaBin(cacheDir); ok {
		return bin, nil
	}

	if err := d.installJavaPackage(ctx, entry.Manifest.URL, cacheDir); err != nil {
		return "", err
	}
	bin, ok := cachedJavaBin(cacheDir)
	if !ok {
		return "", fmt.Errorf("java binary missing after install in %s", cacheDir)
	}
	return bin, nil
}

func componentForJavaMajor(major int) string {
	switch {
	case major >= 21:
		return "java-runtime-delta"
	case major >= 17:
		return "java-runtime-gamma"
	case major >= 16:
		return "java-runtime-alpha"
	case major > 0:
		return "jre-legacy"
	default:
		return ""
	}
}

func javaPlatformKey() string {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "arm64":
			return "windows-arm64"
		case "386":
			return "windows-x86"
		default:
			return "windows-x64"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-os-arm64"
		}
		return "mac-os"
	case "linux":
		if runtime.GOARCH == "386" {
			return "linux-i386"
		}
		return "linux"
	default:
		return runtime.GOOS
	}
}

func cachedJavaBin(dir string) (string, bool) {
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.exe"
	}
	bin := filepath.Join(dir, "bin", name)
	st, err := os.Stat(bin)
	if err != nil || st.IsDir() {
		return "", false
	}
	return bin, true
}

func (d *Downloader) resolveJavaRuntimeEntry(ctx context.Context, platform, component string) (*javaRuntimeEntry, error) {
	body, err := d.fetchBytes(ctx, javaRuntimeCatalogURL)
	if err != nil {
		return nil, fmt.Errorf("java runtime catalog: %w", err)
	}
	var catalog javaRuntimeCatalog
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

func (d *Downloader) installJavaPackage(ctx context.Context, manifestURL, destDir string) error {
	body, err := d.fetchBytes(ctx, manifestURL)
	if err != nil {
		return fmt.Errorf("java package manifest: %w", err)
	}
	var pkg javaPackageManifest
	if err := json.Unmarshal(body, &pkg); err != nil {
		return fmt.Errorf("parse java package manifest: %w", err)
	}
	for rel, file := range pkg.Files {
		if file.Type != "file" {
			continue
		}
		dl := pickJavaDownload(file.Downloads)
		if dl.URL == "" {
			continue
		}
		dest := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := d.downloadIfNeeded(ctx, dl.URL, dest, dl.Sha1); err != nil {
			return fmt.Errorf("java file %s: %w", rel, err)
		}
		if file.Executable {
			_ = os.Chmod(dest, 0o755)
		}
	}
	return nil
}

func pickJavaDownload(downloads map[string]mcmanifest.DownloadFile) mcmanifest.DownloadFile {
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

func (d *Downloader) fetchBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := d.HTTPClient
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
