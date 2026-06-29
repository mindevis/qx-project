package minecraft

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ModelSteve = "steve"
	ModelAlex  = "alex"
)

//go:embed assets/steve.png assets/alex.png
var defaultSkinFS embed.FS

type PlayerSkinConfig struct {
	UUID     string
	Model    string
	SkinPNG  []byte
	SkinURL  string
}

func NormalizeSkinModel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case ModelAlex:
		return ModelAlex
	default:
		return ModelSteve
	}
}

// EnsurePlayerSkin writes the skin PNG into gameDir/skins/{uuid}.png for offline launch.
func EnsurePlayerSkin(gameDir string, cfg PlayerSkinConfig) error {
	if gameDir == "" || cfg.UUID == "" {
		return fmt.Errorf("missing game dir or uuid")
	}
	model := NormalizeSkinModel(cfg.Model)
	skinBytes := cfg.SkinPNG
	if len(skinBytes) == 0 {
		skinName := "steve.png"
		if model == ModelAlex {
			skinName = "alex.png"
		}
		var err error
		skinBytes, err = defaultSkinFS.ReadFile("assets/" + skinName)
		if err != nil {
			return err
		}
	}

	skinDir := filepath.Join(gameDir, "skins")
	if err := os.MkdirAll(skinDir, 0o755); err != nil {
		return err
	}
	skinFile := strings.ReplaceAll(cfg.UUID, "-", "") + ".png"
	skinPath := filepath.Join(skinDir, skinFile)
	return os.WriteFile(skinPath, skinBytes, 0o644)
}

// EnsureOfflineSkin writes the default Steve/Alex skin for an offline profile UUID.
func EnsureOfflineSkin(gameDir, offlineUUID, model string) error {
	return EnsurePlayerSkin(gameDir, PlayerSkinConfig{
		UUID:  offlineUUID,
		Model: model,
	})
}

func DownloadPNG(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 256*1024))
}
