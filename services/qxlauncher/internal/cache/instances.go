package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
)

type InstanceSnapshot struct {
	SyncedAt  time.Time               `json:"synced_at"`
	Instances []apiclient.InstanceItem `json:"instances"`
}

func InstancesPath(dataDir string) string {
	return filepath.Join(dataDir, "instances.json")
}

func SaveInstances(dataDir string, items []apiclient.InstanceItem) error {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	snap := InstanceSnapshot{
		SyncedAt:  time.Now().UTC(),
		Instances: items,
	}
	if snap.Instances == nil {
		snap.Instances = []apiclient.InstanceItem{}
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(InstancesPath(dataDir), b, 0o600)
}

func LoadInstances(dataDir string) (*InstanceSnapshot, error) {
	b, err := os.ReadFile(InstancesPath(dataDir))
	if err != nil {
		return nil, err
	}
	var snap InstanceSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}
