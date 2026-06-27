package minecraft

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ModelSteve = "steve"
	ModelAlex  = "alex"
)

//go:embed assets/steve.png assets/alex.png
var defaultSkinFS embed.FS

type offlineProfileMeta struct {
	UUID     string `json:"uuid"`
	Username string `json:"username,omitempty"`
	Model    string `json:"model"`
	SkinFile string `json:"skin_file"`
}

func NormalizeSkinModel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case ModelAlex:
		return ModelAlex
	default:
		return ModelSteve
	}
}

func EnsureOfflineSkin(gameDir, offlineUUID, model string) error {
	if gameDir == "" || offlineUUID == "" {
		return fmt.Errorf("missing game dir or uuid")
	}
	model = NormalizeSkinModel(model)
	skinName := "steve.png"
	if model == ModelAlex {
		skinName = "alex.png"
	}
	skinBytes, err := defaultSkinFS.ReadFile("assets/" + skinName)
	if err != nil {
		return err
	}

	skinDir := filepath.Join(gameDir, "skins")
	if err := os.MkdirAll(skinDir, 0o755); err != nil {
		return err
	}
	skinFile := strings.ReplaceAll(offlineUUID, "-", "") + ".png"
	skinPath := filepath.Join(skinDir, skinFile)
	if err := os.WriteFile(skinPath, skinBytes, 0o644); err != nil {
		return err
	}

	metaDir := filepath.Join(gameDir, "qx")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return err
	}
	meta := offlineProfileMeta{
		UUID:     offlineUUID,
		Model:    model,
		SkinFile: skinPath,
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	metaPath := filepath.Join(metaDir, strings.ReplaceAll(offlineUUID, "-", "")+".json")
	return os.WriteFile(metaPath, metaBytes, 0o644)
}
