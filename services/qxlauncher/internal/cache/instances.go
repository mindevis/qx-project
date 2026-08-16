package cache

import (
	"encoding/json"
	"os"
	"time"

	"github.com/qxproject/qx/pkg/safepath"
	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
)

type InstanceSnapshot struct {
	SyncedAt  time.Time                `json:"synced_at"`
	Instances []apiclient.InstanceItem `json:"instances"`
}

func InstancesPath(dataDir string) string {
	path, err := safepath.Join(dataDir, "instances.json")
	if err != nil {
		return ""
	}
	return path
}

func InstanceDataRoot(dataDir string) string {
	path, err := safepath.Join(dataDir, "instances")
	if err != nil {
		return ""
	}
	return path
}

// PruneInstanceData removes local instance folders that are no longer on the server.
func PruneInstanceData(dataDir string, items []apiclient.InstanceItem) error {
	root := InstanceDataRoot(dataDir)
	if root == "" {
		return nil
	}
	entries, err := safepath.ReadDir(root)
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
		abs, err := safepath.Join(root, entry.Name())
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := safepath.RemoveAll(abs); err != nil && firstErr == nil {
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
	root, err := safepath.ResolveRoot(dataDir)
	if err != nil {
		return err
	}
	if err := safepath.EnsureDir(root); err != nil {
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
	path, err := safepath.Join(root, "instances.json")
	if err != nil {
		return err
	}
	return safepath.WriteFileBytes(path, b, 0o600)
}

func LoadInstances(dataDir string) (*InstanceSnapshot, error) {
	path := InstancesPath(dataDir)
	if path == "" {
		return nil, os.ErrNotExist
	}
	b, err := safepath.ReadFileBytes(path)
	if err != nil {
		return nil, err
	}
	var snap InstanceSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}
