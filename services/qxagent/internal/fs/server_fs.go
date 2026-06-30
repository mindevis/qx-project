package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

func ReadServerProperties(workDir string) ([]protocol.PropertyEntry, error) {
	path, err := safepath.Join(workDir, "server.properties")
	if err != nil {
		return nil, err
	}
	data, err := safepath.ReadFileBytes(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []protocol.PropertyEntry
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		entry := protocol.PropertyEntry{Key: key, Value: value}
		if isBooleanPropertyValue(value) {
			entry.Boolean = true
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func PatchServerProperties(workDir string, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}
	path, err := safepath.Join(workDir, "server.properties")
	if err != nil {
		return err
	}
	return writePropertyFile(path, updates, nil)
}

func ListDir(workDir, relPath string) ([]protocol.FileEntry, error) {
	abs, err := safepath.JoinRel(workDir, relPath)
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

func ReadFile(workDir, relPath string) (string, int64, error) {
	abs, err := safepath.JoinRel(workDir, relPath)
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
	if info.Size() > 2*1024*1024 {
		return "", 0, fmt.Errorf("file too large")
	}
	data, err := safepath.ReadFileBytes(abs)
	if err != nil {
		return "", 0, err
	}
	return string(data), info.Size(), nil
}

func WriteFile(workDir, relPath, content string) error {
	abs, err := safepath.JoinRel(workDir, relPath)
	if err != nil {
		return err
	}
	if info, err := safepath.Stat(abs); err == nil && info.IsDir() {
		return fmt.Errorf("path is a directory")
	}
	if len(content) > 2*1024*1024 {
		return fmt.Errorf("content too large")
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return err
	}
	return safepath.WriteFileBytes(abs, []byte(content), 0o644)
}

func ListMods(workDir, serverType string) ([]protocol.FileEntry, error) {
	folder := modsFolderFor(serverType)
	if folder == "" {
		return nil, nil
	}
	return ListDir(workDir, folder)
}

func modsFolderFor(serverType string) string {
	switch strings.ToLower(strings.TrimSpace(serverType)) {
	case "paper", "spigot", "purpur", "mohist", "magma", "arclight":
		return "plugins"
	case "forge", "neoforge", "fabric", "quilt":
		return "mods"
	default:
		return ""
	}
}

func isBooleanPropertyValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "false":
		return true
	default:
		return false
	}
}

func writePropertyFile(path string, updates map[string]string, removeKeys []string) error {
	remove := make(map[string]struct{}, len(removeKeys))
	for _, key := range removeKeys {
		remove[key] = struct{}{}
	}
	lines := make([]string, 0, 32)
	if data, err := safepath.ReadFileBytes(path); err == nil {
		lines = strings.Split(string(data), "\n")
	} else if !os.IsNotExist(err) {
		return err
	}

	seen := make(map[string]struct{}, len(updates))
	out := make([]string, 0, len(lines)+len(updates))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			out = append(out, line)
			continue
		}
		key = strings.TrimSpace(key)
		if _, drop := remove[key]; drop {
			continue
		}
		if value, ok := updates[key]; ok {
			out = append(out, fmt.Sprintf("%s=%s", key, value))
			seen[key] = struct{}{}
			continue
		}
		out = append(out, line)
	}
	for key, value := range updates {
		if _, ok := seen[key]; ok {
			continue
		}
		out = append(out, fmt.Sprintf("%s=%s", key, value))
	}
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return safepath.WriteFileBytes(path, []byte(content), 0o644)
}
