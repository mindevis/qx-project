package device

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func DeviceIDPath(dataDir string) string {
	return filepath.Join(dataDir, "device_id")
}

func LoadDeviceID(dataDir string) string {
	if id := strings.TrimSpace(os.Getenv("QX_DEVICE_ID")); id != "" {
		return id
	}
	b, err := os.ReadFile(DeviceIDPath(dataDir))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func SaveDeviceID(dataDir, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(DeviceIDPath(dataDir), []byte(deviceID), 0o600)
}

func ResolveDeviceID(dataDir string) string {
	if id := LoadDeviceID(dataDir); id != "" {
		return id
	}
	return uuid.NewString()
}

func ReadToken(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
