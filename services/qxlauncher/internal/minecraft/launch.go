package minecraft

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type Downloader struct {
	RootDir    string
	AssetsCDN  string
	HTTPClient *http.Client
}

func NewDownloader(root string) *Downloader {
	return &Downloader{
		RootDir:    root,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

func (d *Downloader) EnsureClientJar(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if manifest == nil || manifest.ClientJar.URL == "" {
		return "", fmt.Errorf("missing client jar")
	}
	dir := filepath.Join(d.RootDir, "instances", manifest.InstanceID, "versions", manifest.MCVersion)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, manifest.MCVersion+".jar")
	return dest, d.downloadIfNeeded(ctx, manifest.ClientJar.URL, dest, manifest.ClientJar.Sha1)
}

func (d *Downloader) downloadIfNeeded(ctx context.Context, url, dest, sha1hex string) error {
	if sha1hex != "" {
		if b, err := os.ReadFile(dest); err == nil {
			if hex.EncodeToString(sha1Sum(b)) == strings.ToLower(sha1hex) {
				return nil
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", url, res.StatusCode)
	}
	tmp := dest + ".part"
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, res.Body)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dest)
}

func sha1Sum(b []byte) []byte {
	sum := sha1.Sum(b)
	return sum[:]
}

type LaunchPlan struct {
	JavaBin    string
	Args       []string
	MainClass  string
	WorkingDir string
}

func BuildLaunchPlan(manifest *mcmanifest.InstanceLaunchManifest, clientJar string, libPaths []string, nativesDir, assetsDir, gameDir, username, offlineUUID, javaBin string) LaunchPlan {
	if gameDir == "" {
		gameDir = filepath.Dir(clientJar)
	}
	if assetsDir == "" {
		assetsDir = filepath.Join(gameDir, "assets")
	}
	cpEntries := append(append([]string{}, libPaths...), clientJar)
	args := []string{"-Xmx2G"}
	if nativesDir != "" {
		args = append(args, "-Djava.library.path="+nativesDir)
	}
	args = append(args,
		"-cp", BuildClasspath(cpEntries),
		manifest.MainClass,
		"--username", username,
		"--version", manifest.MCVersion,
		"--gameDir", gameDir,
		"--assetsDir", assetsDir,
		"--assetIndex", manifest.AssetIndex.ID,
		"--uuid", offlineUUID,
		"--accessToken", "0",
		"--userType", "legacy",
	)
	if javaBin == "" {
		javaBin = ResolveJavaBin()
	}
	return LaunchPlan{
		JavaBin:    javaBin,
		Args:       args,
		MainClass:  manifest.MainClass,
		WorkingDir: gameDir,
	}
}
