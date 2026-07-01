package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
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
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.EvalSymlinks(current)
	if err != nil {
		return err
	}

	stagingDir := stagingDir()
	if err := os.MkdirAll(stagingDir, 0o700); err != nil {
		return err
	}
	staging := filepath.Join(stagingDir, filename+".staging")

	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("download failed: %d %s", res.StatusCode, string(b))
	}

	out, err := os.OpenFile(staging, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, res.Body); err != nil {
		out.Close()
		_ = os.Remove(staging)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(staging)
		return err
	}

	if err := replaceExecutable(current, staging); err != nil {
		_ = os.Remove(staging)
		return err
	}
	_ = os.Remove(staging)

	cmd := proc.Command(current)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
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
	_ = os.Remove(backup)

	if err := os.Rename(current, backup); err != nil {
		return fmt.Errorf("rename running executable: %w", err)
	}
	if err := copyFile(staging, current); err != nil {
		_ = os.Rename(backup, current)
		return fmt.Errorf("install update: %w", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}
