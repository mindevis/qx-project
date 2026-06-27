package minecraft

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func (d *Downloader) EnsureLoaderInstalled(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest, javaBin string) error {
	if manifest == nil {
		return nil
	}
	switch manifest.Loader {
	case mcmanifest.LoaderForge, mcmanifest.LoaderNeoForge:
	default:
		return nil
	}
	if manifest.LoaderClientJar.RelativePath == "" {
		manifest.LoaderClientJar = mcmanifest.DefaultLoaderClientJar(manifest.Loader, manifest.MCVersion, manifest.LoaderVersion)
	}
	if manifest.LoaderClientJar.RelativePath == "" {
		return fmt.Errorf("%s manifest missing loader client jar metadata", manifest.Loader)
	}
	dest := filepath.Join(d.RootDir, filepath.FromSlash(manifest.LoaderClientJar.RelativePath))
	if jarMatches(dest, manifest.LoaderClientJar.Sha1) {
		return nil
	}
	if javaBin == "" {
		javaBin = ResolveJavaBin("")
	}
	d.progress("loader-install", fmt.Sprintf("running %s installer (first run can take a few minutes) …", manifest.Loader))
	return d.runLoaderInstaller(ctx, manifest, javaBin)
}

func jarMatches(path, sha1hex string) bool {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return false
	}
	if sha1hex == "" {
		return true
	}
	return strings.EqualFold(hex.EncodeToString(sha1Sum(b)), sha1hex)
}

func (d *Downloader) runLoaderInstaller(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest, javaBin string) error {
	cacheDir := filepath.Join(d.RootDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	installerPath := filepath.Join(cacheDir, "installer-"+manifest.VersionID+".jar")
	if err := d.downloadIfNeeded(ctx, manifest.VersionURL, installerPath, ""); err != nil {
		return fmt.Errorf("download installer: %w", err)
	}
	profilesPath := filepath.Join(d.RootDir, "launcher_profiles.json")
	if err := os.WriteFile(profilesPath, []byte("{}"), 0o644); err != nil {
		return fmt.Errorf("write launcher_profiles.json: %w", err)
	}
	cmd := exec.CommandContext(ctx, javaBin, "-jar", installerPath, "--installClient", d.RootDir)
	cmd.Dir = d.RootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		tail := string(out)
		if len(tail) > 4000 {
			tail = tail[len(tail)-4000:]
		}
		return fmt.Errorf("installer failed: %w\n%s", err, tail)
	}
	dest := filepath.Join(d.RootDir, filepath.FromSlash(manifest.LoaderClientJar.RelativePath))
	if !jarMatches(dest, manifest.LoaderClientJar.Sha1) {
		return fmt.Errorf("installer finished but %s is missing or corrupt", dest)
	}
	return nil
}
