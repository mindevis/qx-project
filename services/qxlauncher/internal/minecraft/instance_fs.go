package minecraft

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

const instanceFileMaxBytes = 2 * 1024 * 1024
const maxResourceUploadBytes = protocol.MaxContentFileBytes

func (d *Downloader) ListInstanceDir(instanceID, relPath string) ([]protocol.FileEntry, error) {
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return nil, err
	}
	abs, err := safepath.JoinRel(gameDir, relPath)
	if err != nil {
		return nil, err
	}
	entries, err := safepath.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.FileEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		childRel := name
		if relPath != "" {
			childRel = filepath.ToSlash(filepath.Join(relPath, name))
		}
		item := protocol.FileEntry{
			Name: name,
			Path: childRel,
			Dir:  entry.IsDir(),
		}
		if !entry.IsDir() {
			item.Size = info.Size()
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (d *Downloader) ReadInstanceFile(instanceID, relPath string) (string, int64, error) {
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return "", 0, err
	}
	abs, err := safepath.JoinRel(gameDir, relPath)
	if err != nil {
		return "", 0, err
	}
	info, err := safepath.Stat(abs)
	if err != nil {
		return "", 0, err
	}
	if info.IsDir() {
		return "", 0, fmt.Errorf("path is a directory")
	}
	if info.Size() > instanceFileMaxBytes {
		return "", 0, fmt.Errorf("file too large")
	}
	data, err := safepath.ReadFileBytes(abs)
	if err != nil {
		return "", 0, err
	}
	return string(data), info.Size(), nil
}

func (d *Downloader) WriteInstanceFile(instanceID, relPath, content string) error {
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return err
	}
	abs, err := safepath.JoinRel(gameDir, relPath)
	if err != nil {
		return err
	}
	if info, err := safepath.Stat(abs); err == nil && info.IsDir() {
		return fmt.Errorf("path is a directory")
	}
	if len(content) > instanceFileMaxBytes {
		return fmt.Errorf("content too large")
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return err
	}
	return safepath.WriteFileBytes(abs, []byte(content), 0o644)
}

func (d *Downloader) ReadInstanceResourceFile(instanceID, folder, filename string) ([]byte, error) {
	if instanceID == "" || folder == "" || filename == "" {
		return nil, fmt.Errorf("invalid read parameters")
	}
	relPath := filepath.ToSlash(filepath.Join(folder, filename))
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return nil, err
	}
	abs, err := safepath.JoinRel(gameDir, relPath)
	if err != nil {
		return nil, err
	}
	info, err := safepath.Stat(abs)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > maxResourceUploadBytes {
		return nil, fmt.Errorf("file too large")
	}
	return safepath.ReadFileBytes(abs)
}

func (d *Downloader) WriteInstanceResourceFile(instanceID, folder, filename string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("invalid upload parameters")
	}
	return d.WriteInstanceResourceStream(instanceID, folder, filename, bytes.NewReader(data))
}

func (d *Downloader) WriteInstanceResourceStream(instanceID, folder, filename string, r io.Reader) error {
	if instanceID == "" || folder == "" || filename == "" || r == nil {
		return fmt.Errorf("invalid upload parameters")
	}
	relPath := filepath.ToSlash(filepath.Join(folder, filename))
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return err
	}
	abs, err := safepath.JoinRel(gameDir, relPath)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return err
	}
	return safepath.WriteStreamAtomic(abs, r)
}

func (d *Downloader) RemoveInstanceResourceFile(instanceID, folder, filename string) error {
	if instanceID == "" || folder == "" || filename == "" {
		return fmt.Errorf("invalid uninstall parameters")
	}
	relPath := filepath.ToSlash(filepath.Join(folder, filename))
	gameDir, err := d.InstanceGameDir(instanceID)
	if err != nil {
		return err
	}
	abs, err := safepath.JoinRel(gameDir, relPath)
	if err != nil {
		return err
	}
	if err := safepath.Remove(abs); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func IsConfigExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".toml", ".json", ".properties", ".cfg", ".yml", ".yaml", ".txt":
		return true
	default:
		return false
	}
}
