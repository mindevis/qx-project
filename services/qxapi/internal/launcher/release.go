package launcher

import (
	"path"
	"strings"
)

// ReleaseInfo describes the latest published QXLauncher build.
type ReleaseInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
}

func (s *Service) releaseInfo() ReleaseInfo {
	version := strings.TrimSpace(s.launcherVersion)
	if version == "" {
		version = "0.1.0-dev"
	}
	downloadURL := absoluteReleaseURL(s.webBaseURL, s.launcherDownloadURL)
	filename := path.Base(downloadURL)
	if filename == "" || filename == "." || filename == "/" {
		filename = "qx-launcher.exe"
	}
	return ReleaseInfo{
		Version:     version,
		DownloadURL: downloadURL,
		Filename:    filename,
	}
}

func absoluteReleaseURL(base, raw string) string {
	raw = strings.TrimSpace(raw)
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if raw == "" {
		raw = "/downloads/qx-launcher.exe"
	}
	if strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
		return raw
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	if base == "" {
		return raw
	}
	return base + raw
}

func (s *Service) GetRelease() ReleaseInfo {
	return s.releaseInfo()
}
