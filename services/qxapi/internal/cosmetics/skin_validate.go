package cosmetics

import (
	"bytes"
	"errors"
	"image"
	_ "image/png"
)

var (
	ErrInvalidSkinFormat = errors.New("skin must be a PNG image")
	ErrInvalidSkinSize   = errors.New("skin must be 64x64 or 64x32 pixels")
	ErrSkinTooLarge      = errors.New("skin file too large")
)

const maxSkinBytes = 256 * 1024

func ValidateSkinPNG(data []byte) error {
	if len(data) == 0 {
		return ErrInvalidSkinFormat
	}
	if len(data) > maxSkinBytes {
		return ErrSkinTooLarge
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" {
		return ErrInvalidSkinFormat
	}
	if (cfg.Width == 64 && (cfg.Height == 64 || cfg.Height == 32)) ||
		(cfg.Width == 128 && cfg.Height == 128) {
		return nil
	}
	return ErrInvalidSkinSize
}
