package fs

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

const contentDownloadTimeout = 5 * time.Minute
const contentDownloadUserAgent = "QXProject/1.0 (https://github.com/qxproject/qx)"

func ListPlugins(workDir string) ([]protocol.FileEntry, error) {
	return ListDir(workDir, "plugins")
}

func ListDatapacks(workDir string) ([]protocol.FileEntry, error) {
	world, err := WorldFolder(workDir)
	if err != nil {
		return nil, err
	}
	return ListDir(workDir, filepath.Join(world, "datapacks"))
}

func WorldFolder(workDir string) (string, error) {
	levelName := "world"
	props, err := ReadServerProperties(workDir)
	if err != nil {
		return "", err
	}
	for _, entry := range props {
		if entry.Key == "level-name" && strings.TrimSpace(entry.Value) != "" {
			levelName = strings.TrimSpace(entry.Value)
			break
		}
	}
	if strings.Contains(levelName, "..") || strings.ContainsAny(levelName, `/\`) {
		return "", fmt.Errorf("invalid level-name")
	}
	return levelName, nil
}

func ContentRelPath(workDir, serverType, contentKind, filename, modTarget string) (string, error) {
	filename = strings.TrimSpace(filename)
	if filename == "" || strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\`) {
		return "", fmt.Errorf("invalid filename")
	}
	switch strings.ToLower(strings.TrimSpace(contentKind)) {
	case "mod":
		folder := modFolderFor(serverType, modTarget)
		if folder == "" {
			return "", fmt.Errorf("server type does not support mods")
		}
		return filepath.ToSlash(filepath.Join(folder, filename)), nil
	case "plugin":
		return filepath.ToSlash(filepath.Join("plugins", filename)), nil
	case "datapack":
		world, err := WorldFolder(workDir)
		if err != nil {
			return "", err
		}
		return filepath.ToSlash(filepath.Join(world, "datapacks", filename)), nil
	case "resourcepack":
		folder := resourcepackFolderFor(modTarget)
		if folder == "" {
			return "", fmt.Errorf("invalid resourcepack target")
		}
		return filepath.ToSlash(filepath.Join(folder, filename)), nil
	case "shader":
		folder := shaderFolderFor(modTarget)
		if folder == "" {
			return "", fmt.Errorf("invalid shader target")
		}
		return filepath.ToSlash(filepath.Join(folder, filename)), nil
	default:
		return "", fmt.Errorf("unknown content kind %q", contentKind)
	}
}

// extraContentDownloadHosts is for tests (httptest). Production stays on the catalog CDN list.
var extraContentDownloadHosts []string

var allowedContentDownloadHosts = []string{
	"cdn.modrinth.com",
	"cdn-raw.modrinth.com",
	"hangarcdn.papermc.io",
	"api.spiget.org",
	"cdn.spiget.org",
	"edge.forgecdn.net",
	"mediafilez.forgecdn.net",
	"media.forgecdn.net",
	"github.com",
}

var allowedContentDownloadHostSuffixes = []string{
	"modrinth.com",
	"papermc.io",
	"spiget.org",
	"forgecdn.net",
	"cursecdn.com",
	"curseforge.com",
	"githubusercontent.com",
}

func isBlockedCustomDownloadHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || net.ParseIP(host) != nil {
		return true
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || strings.HasSuffix(host, ".lan") {
		return true
	}
	return false
}

func hostAllowedForDownload(host string, allowCustomHost bool) bool {
	if allowCustomHost {
		return !isBlockedCustomDownloadHost(host)
	}
	return isAllowedContentDownloadHost(host)
}

func isAllowedContentDownloadHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || net.ParseIP(host) != nil {
		return false
	}
	for _, allowed := range allowedContentDownloadHosts {
		if host == allowed {
			return true
		}
	}
	for _, suffix := range allowedContentDownloadHostSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	for _, extra := range extraContentDownloadHosts {
		if host == extra {
			return true
		}
	}
	return false
}

// sanitizeContentDownloadURL rejects SSRF targets and returns a reconstructed https URL.
func sanitizeContentDownloadURL(raw string) (string, error) {
	return sanitizeDownloadURL(raw, false)
}

func sanitizeUserContentDownloadURL(raw string) (string, error) {
	return sanitizeDownloadURL(raw, true)
}

func sanitizeDownloadURL(raw string, allowCustomHost bool) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid download url")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("invalid download url")
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if len(extraContentDownloadHosts) == 0 && scheme != "https" {
		return "", fmt.Errorf("download url must be https")
	}
	if scheme != "https" && scheme != "http" {
		return "", fmt.Errorf("invalid download url")
	}
	if !hostAllowedForDownload(host, allowCustomHost) {
		return "", fmt.Errorf("download host not allowed")
	}
	// Parse decodes once; a second unescape fixes already-double-encoded
	// names like TAB%2520v6.1.2.jar. Do not copy RawPath — it preserves %2520.
	path := parsed.Path
	if strings.Contains(path, "%") {
		if decoded, err := url.PathUnescape(path); err == nil {
			path = decoded
		}
	}
	safe := &url.URL{
		Scheme:   scheme,
		Host:     parsed.Host,
		Path:     path,
		RawQuery: parsed.RawQuery,
	}
	if safe.Path == "" {
		safe.Path = "/"
	}
	return safe.String(), nil
}

func InstallContentFile(ctx context.Context, workDir, relPath, downloadURL string, allowCustomHost bool) error {
	relPath = strings.TrimSpace(relPath)
	downloadURL = strings.TrimSpace(downloadURL)
	if relPath == "" || downloadURL == "" {
		return fmt.Errorf("rel_path and download_url required")
	}
	var err error
	if allowCustomHost {
		downloadURL, err = sanitizeUserContentDownloadURL(downloadURL)
	} else {
		downloadURL, err = sanitizeContentDownloadURL(downloadURL)
	}
	if err != nil {
		return err
	}
	abs, err := safepath.JoinRel(workDir, relPath)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	downloadCtx, cancel := context.WithTimeout(ctx, contentDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", contentDownloadUserAgent)
	client := &http.Client{
		Timeout: 0,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			if !hostAllowedForDownload(req.URL.Hostname(), allowCustomHost) {
				return fmt.Errorf("download host not allowed")
			}
			req.Header.Set("User-Agent", contentDownloadUserAgent)
			return nil
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", downloadURL, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: http %d", downloadURL, res.StatusCode)
	}
	return safepath.WriteStreamAtomic(abs, res.Body)
}

func UploadContentFile(workDir, serverType, contentKind, modTarget, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty content")
	}
	if len(data) > protocol.MaxContentFileBytes {
		return "", fmt.Errorf("content too large")
	}
	relPath, err := ContentRelPath(workDir, serverType, contentKind, filename, modTarget)
	if err != nil {
		return "", err
	}
	abs, err := safepath.JoinRel(workDir, relPath)
	if err != nil {
		return "", err
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return "", err
	}
	if err := safepath.WriteFileBytes(abs, data, 0o644); err != nil {
		return "", err
	}
	return relPath, nil
}

func DeleteContentFile(workDir, serverType, contentKind, modTarget, filename string) (string, error) {
	relPath, err := ContentRelPath(workDir, serverType, contentKind, filename, modTarget)
	if err != nil {
		return "", err
	}
	abs, err := safepath.JoinRel(workDir, relPath)
	if err != nil {
		return "", err
	}
	if err := safepath.Remove(abs); err != nil {
		return "", err
	}
	return relPath, nil
}

func ReadContentFile(workDir, serverType, contentKind, modTarget, filename string) ([]byte, error) {
	relPath, err := ContentRelPath(workDir, serverType, contentKind, filename, modTarget)
	if err != nil {
		return nil, err
	}
	abs, err := safepath.JoinRel(workDir, relPath)
	if err != nil {
		return nil, err
	}
	info, err := safepath.Stat(abs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > protocol.MaxContentFileBytes {
		return nil, fmt.Errorf("content too large: %s is %d bytes (limit %d)", filename, info.Size(), protocol.MaxContentFileBytes)
	}
	return safepath.ReadFileBytes(abs)
}
