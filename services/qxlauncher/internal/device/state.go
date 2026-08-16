package device

import (
	"strings"

	"github.com/google/uuid"

	"github.com/qxproject/qx/pkg/safepath"
)

func DeviceIDPath(dataDir string) string {
	path, err := safepath.Join(dataDir, "device_id")
	if err != nil {
		return ""
	}
	return path
}

func LoadDeviceID(dataDir string) string {
	path := DeviceIDPath(dataDir)
	if path == "" {
		return ""
	}
	b, err := safepath.ReadFileBytes(path)
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
	root, err := safepath.ResolveRoot(dataDir)
	if err != nil {
		return err
	}
	if err := safepath.EnsureDir(root); err != nil {
		return err
	}
	path, err := safepath.Join(root, "device_id")
	if err != nil {
		return err
	}
	return safepath.WriteFileBytes(path, []byte(deviceID), 0o600)
}

func ResolveDeviceID(dataDir string) string {
	if id := LoadDeviceID(dataDir); id != "" {
		return id
	}
	if id := MachineDeviceID(); id != "" {
		_ = SaveDeviceID(dataDir, id)
		return id
	}
	id := uuid.NewString()
	_ = SaveDeviceID(dataDir, id)
	return id
}

func ReadToken(path string) string {
	abs, err := safepath.ResolveRoot(path)
	if err != nil {
		return ""
	}
	b, err := safepath.ReadFileBytes(abs)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
