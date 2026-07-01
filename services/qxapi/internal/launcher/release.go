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
	downloadURL := strings.TrimSpace(s.launcherDownloadURL)
	if downloadURL == "" {
		downloadURL = s.webBaseURL + "/downloads/qx-launcher.exe"
	}
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

func (s *Service) GetRelease() ReleaseInfo {
	return s.releaseInfo()
}
