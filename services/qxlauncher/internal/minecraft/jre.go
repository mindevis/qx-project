package minecraft

import (
	"context"
	"path/filepath"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/pkg/mojangjava"
)

// EnsureJava returns a Mojang-provided Java binary for the launch manifest.
// EnsureJRE uses Mojang JRE download unless java_path (.env QX_JAVA / launcher.toml) or skip_java_download is set.
func (d *Downloader) EnsureJava(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	return d.javaManager().EnsureForManifest(ctx, manifest)
}

func (d *Downloader) javaManager() *mojangjava.Manager {
	return &mojangjava.Manager{
		RootDir:      filepath.Join(d.RootDir, "java"),
		JavaPath:     d.JavaPath,
		SkipDownload: d.SkipJavaDownload,
		HTTPClient:   d.HTTPClient,
	}
}

func componentForJavaMajor(major int) string {
	return mojangjava.ComponentForMajor(major)
}

func javaPlatformKey() string {
	return mojangjava.PlatformKey()
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
