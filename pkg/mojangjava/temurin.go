package mojangjava

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const temurinUserAgent = "QXProject/1.0 (https://github.com/qxproject/qx)"

func (m *Manager) ensureTemurin(ctx context.Context, major int) (string, error) {
	if major <= 0 {
		return "", fmt.Errorf("java major required")
	}
	cacheDir := filepath.Join(m.RootDir, "temurin", fmt.Sprintf("%d", major))
	if bin, ok := findJavaBin(cacheDir); ok {
		return bin, nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir temurin: %w", err)
	}

	url := strings.TrimSpace(m.TemurinURL)
	if url == "" {
		url = temurinLatestURL(major)
	}
	archive := filepath.Join(m.RootDir, fmt.Sprintf("temurin-%d-download", major))
	kind, err := m.downloadTemurinArchive(ctx, url, archive)
	if err != nil {
		return "", err
	}
	defer os.Remove(archive)

	switch kind {
	case "zip":
		if err := extractZip(archive, cacheDir); err != nil {
			return "", fmt.Errorf("extract temurin zip: %w", err)
		}
	default:
		if err := extractTarGz(archive, cacheDir); err != nil {
			return "", fmt.Errorf("extract temurin tgz: %w", err)
		}
	}
	bin, ok := findJavaBin(cacheDir)
	if !ok {
		return "", fmt.Errorf("temurin java %d binary missing after install in %s", major, cacheDir)
	}
	return bin, nil
}

func temurinLatestURL(major int) string {
	return fmt.Sprintf(
		"https://api.adoptium.net/v3/binary/latest/%d/ga/%s/%s/jdk/hotspot/normal/eclipse?project=jdk",
		major,
		temurinOS(),
		temurinArch(),
	)
}

func temurinOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "mac"
	default:
		return "linux"
	}
}

func temurinArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "aarch64"
	case "386":
		return "x86"
	default:
		return "x64"
	}
}

func findJavaBin(root string) (string, bool) {
	candidates := []string{root}
	entries, err := os.ReadDir(root)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			sub := filepath.Join(root, entry.Name())
			candidates = append(candidates, sub, filepath.Join(sub, "Contents", "Home"))
		}
	}
	for _, dir := range candidates {
		if bin, ok := CachedJavaBin(dir); ok {
			return bin, true
		}
	}
	return "", false
}

func (m *Manager) downloadTemurinArchive(ctx context.Context, url, dest string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", temurinUserAgent)
	client := m.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download temurin: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download temurin: http %d", res.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(out, res.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return archiveKind(dest, res.Header.Get("Content-Type"), res.Request.URL.Path), nil
}

func archiveKind(path, contentType, urlPath string) string {
	lower := strings.ToLower(contentType + " " + urlPath + " " + path)
	if strings.Contains(lower, "gzip") || strings.Contains(lower, ".tgz") || strings.Contains(lower, ".tar.gz") || strings.Contains(lower, "x-tar") {
		return "tgz"
	}
	if strings.Contains(lower, "zip") {
		return "zip"
	}
	if runtime.GOOS == "windows" {
		return "zip"
	}
	return "tgz"
}

func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(hdr.Name)
		if name == "" || strings.Contains(name, "..") {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode)
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}

func extractZip(archive, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, file := range zr.File {
		name := filepath.ToSlash(file.Name)
		if name == "" || strings.Contains(name, "..") {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeOut := out.Close()
		closeRC := rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeOut != nil {
			return closeOut
		}
		if closeRC != nil {
			return closeRC
		}
	}
	return nil
}
