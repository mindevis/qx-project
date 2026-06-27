package mojangjava

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type Manager struct {
	RootDir      string
	JavaPath     string
	SkipDownload bool
	HTTPClient   *http.Client
	CatalogURL   string
}

func (m *Manager) EnsureForMcVersion(ctx context.Context, mcVersion string) (string, error) {
	if custom := strings.TrimSpace(m.JavaPath); custom != "" {
		return custom, nil
	}
	if m.SkipDownload {
		return ResolveSystemJava(""), nil
	}
	mcVersion = strings.TrimSpace(mcVersion)
	if mcVersion == "" {
		return "", fmt.Errorf("mc version required for java install")
	}

	client := mcmanifest.NewClient()
	if m.HTTPClient != nil {
		client.HTTPClient = m.HTTPClient
	}
	versionURL, err := client.ResolveVersionURL(ctx, mcVersion)
	if err != nil {
		return "", err
	}
	meta, err := client.FetchVersionMeta(ctx, versionURL)
	if err != nil {
		return "", err
	}
	return m.EnsureForRuntime(ctx, meta.JavaVersion.Component, meta.JavaVersion.MajorVersion)
}

func (m *Manager) EnsureForManifest(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if custom := strings.TrimSpace(m.JavaPath); custom != "" {
		return custom, nil
	}
	if m.SkipDownload {
		return ResolveSystemJava(""), nil
	}
	if manifest == nil {
		return "", fmt.Errorf("missing manifest")
	}
	component := strings.TrimSpace(manifest.JavaComponent)
	if component == "" {
		component = ComponentForMajor(manifest.JavaMajor)
	}
	if component == "" {
		return "", fmt.Errorf("unknown java component for major %d", manifest.JavaMajor)
	}
	return m.EnsureForRuntime(ctx, component, manifest.JavaMajor)
}

func (m *Manager) EnsureForRuntime(ctx context.Context, component string, major int) (string, error) {
	if custom := strings.TrimSpace(m.JavaPath); custom != "" {
		return custom, nil
	}
	if m.SkipDownload {
		return ResolveSystemJava(""), nil
	}
	component = strings.TrimSpace(component)
	if component == "" {
		component = ComponentForMajor(major)
	}
	if component == "" {
		return "", fmt.Errorf("unknown java component for major %d", major)
	}
	if strings.TrimSpace(m.RootDir) == "" {
		return "", fmt.Errorf("java root dir is required")
	}

	platform := PlatformKey()
	entry, err := m.resolveRuntimeEntry(ctx, platform, component)
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(m.RootDir, component, entry.Version.Name)
	if bin, ok := CachedJavaBin(cacheDir); ok {
		return bin, nil
	}
	if err := m.installPackage(ctx, entry.Manifest.URL, cacheDir); err != nil {
		return "", err
	}
	bin, ok := CachedJavaBin(cacheDir)
	if !ok {
		return "", fmt.Errorf("java binary missing after install in %s", cacheDir)
	}
	return bin, nil
}
