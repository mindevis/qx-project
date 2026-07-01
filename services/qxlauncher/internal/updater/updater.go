package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Apply downloads the launcher binary and schedules replacement of the running executable.
// On Windows the current process exits after spawning a helper batch script.
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

	dir := filepath.Dir(current)
	staging := filepath.Join(dir, ".qxlauncher-update-"+filename)
	out, err := os.Create(staging)
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

	scriptPath := filepath.Join(os.TempDir(), "qxlauncher-update.cmd")
	script := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /y "%s" "%s" >nul
start "" "%s"
del "%%~f0"
`, staging, current, current)
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		_ = os.Remove(staging)
		return err
	}
	cmd := exec.Command("cmd", "/C", scriptPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		_ = os.Remove(staging)
		_ = os.Remove(scriptPath)
		return err
	}
	os.Exit(0)
	return nil
}
