package updater

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/safepath"
	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
	"github.com/qxproject/qx/services/qxlauncher/internal/version"
)

const backupSuffix = ".prev"

// Apply downloads the launcher binary and replaces the running executable.
// On Windows the running binary is renamed in place (no helper script), a new
// copy is written to the original path, and the updated binary is started.
func Apply(ctx context.Context, downloadURL, filename string, httpClient *http.Client) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("self-update is only supported on Windows")
	}
	if downloadURL == "" {
		return fmt.Errorf("download url is empty")
	}
	if filename == "" {
		filename = "qx-launcher.exe"
	}
	if strings.ContainsAny(filename, `/\`) || strings.Contains(filename, "..") {
		return fmt.Errorf("invalid update filename")
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.EvalSymlinks(current)
	if err != nil {
		return err
	}

	dir, err := safepath.ResolveRoot(stagingDir())
	if err != nil {
		return err
	}
	if err := safepath.EnsureDir(dir); err != nil {
		return err
	}
	staging, err := safepath.Join(dir, filename+".staging")
	if err != nil {
		return err
	}

	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "QXLauncher/"+version.Version)
	slog.Info("launcher update download", "url", downloadURL, "staging", staging)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("download failed: %d %s", res.StatusCode, string(b))
	}

	out, err := safepath.OpenFile(staging, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, res.Body)
	if err != nil {
		out.Close()
		_ = safepath.Remove(staging)
		return err
	}
	if err := out.Close(); err != nil {
		_ = safepath.Remove(staging)
		return err
	}
	slog.Info("launcher update downloaded", "bytes", n, "status", res.StatusCode)
	if err := validateWindowsExecutable(staging); err != nil {
		_ = safepath.Remove(staging)
		return err
	}

	if err := replaceExecutable(current, staging); err != nil {
		_ = safepath.Remove(staging)
		return err
	}
	_ = safepath.Remove(staging)
	slog.Info("launcher update installed", "exe", current)
	return nil
}

// ResolveURL turns a site-relative download path into an absolute URL using
// the scheme and host of base (typically the launcher API root).
func ResolveURL(base, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
		return raw
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return raw
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	return parsed.Scheme + "://" + parsed.Host + raw
}

// Restart starts a new process of the current executable. The caller should exit.
func Restart() error {
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.EvalSymlinks(current)
	if err != nil {
		return err
	}
	cmd := proc.Command(current)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func stagingDir() string {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "QXLauncher", "updates")
	}
	return filepath.Join(os.TempDir(), "QXLauncher", "updates")
}

func backupPath(exe string) string {
	return exe + backupSuffix
}

func replaceExecutable(current, staging string) error {
	backup := backupPath(current)
	_ = safepath.Remove(backup)

	if err := safepath.Rename(current, backup); err != nil {
		return fmt.Errorf("rename running executable: %w", err)
	}
	if err := copyFile(staging, current); err != nil {
		_ = safepath.Rename(backup, current)
		return fmt.Errorf("install update: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := safepath.OpenRead(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := safepath.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = safepath.Remove(dst)
		return err
	}
	return out.Close()
}
