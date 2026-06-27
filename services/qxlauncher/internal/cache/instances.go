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

func InstanceDataRoot(dataDir string) string {
	return filepath.Join(dataDir, "instances")
}

// PruneInstanceData removes local instance folders that are no longer on the server.
func PruneInstanceData(dataDir string, items []apiclient.InstanceItem) error {
	root := InstanceDataRoot(dataDir)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	keep := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.ID != "" {
			keep[item.ID] = struct{}{}
		}
	}

	var firstErr error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, ok := keep[entry.Name()]; ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// SyncInstances updates the local cache and deletes instance data removed on the website.
func SyncInstances(dataDir string, items []apiclient.InstanceItem) error {
	_ = PruneInstanceData(dataDir, items)
	return SaveInstances(dataDir, items)
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
