package servers

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

const (
	clientConfigRoot   = "client-config"
	instanceConfigRoot = "config"
	maxConfigWalkDepth = 3
	maxConfigFileBytes = 2 * 1024 * 1024
)

func isConfigFilePath(path string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".toml", ".json", ".properties", ".cfg", ".yml", ".yaml":
		return true
	default:
		return false
	}
}

func configRelativePath(path string) string {
	p := strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	p = strings.TrimPrefix(p, "/")
	lower := strings.ToLower(p)
	switch {
	case strings.HasPrefix(lower, "client-config/"):
		return p[len("client-config/"):]
	case lower == "client-config":
		return ""
	case strings.HasPrefix(lower, "config/"):
		return p[len("config/"):]
	case lower == "config":
		return ""
	default:
		return p
	}
}

func instanceConfigDestPath(serverPath string) string {
	rel := strings.Trim(configRelativePath(serverPath), "/")
	if rel == "" {
		return instanceConfigRoot
	}
	return instanceConfigRoot + "/" + rel
}

func (s *Service) walkGameServerConfigFiles(
	ctx context.Context,
	ownerID, vpsID, gameServerID, root string,
	maxDepth int,
) ([]protocol.FileEntry, error) {
	var files []protocol.FileEntry
	var walk func(path string, depth int) error
	walk = func(path string, depth int) error {
		entries, err := s.ListGameServerFiles(ctx, ownerID, vpsID, gameServerID, path)
		if err != nil {
			if depth == 0 {
				return err
			}
			return nil
		}
		for _, entry := range entries {
			if entry.Dir {
				if depth < maxDepth {
					if err := walk(entry.Path, depth+1); err != nil {
						return err
					}
				}
				continue
			}
			if isConfigFilePath(entry.Path) {
				files = append(files, entry)
			}
		}
		return nil
	}
	if err := walk(root, 0); err != nil {
		return nil, err
	}
	return files, nil
}

func (s *Service) pullClientConfigsToInstance(
	ctx context.Context,
	ownerID, vpsID, gameServerID, instanceID string,
	launcherSvc *launcher.Service,
	owner launcher.Owner,
	result *PrepareConnectModsResult,
) {
	files, err := s.walkGameServerConfigFiles(ctx, ownerID, vpsID, gameServerID, clientConfigRoot, maxConfigWalkDepth)
	if err != nil {
		result.Errors = append(result.Errors, clientConfigRoot+": "+err.Error())
		return
	}
	for _, entry := range files {
		dest := instanceConfigDestPath(entry.Path)
		read, readErr := s.ReadGameServerFile(ctx, ownerID, vpsID, gameServerID, entry.Path)
		if readErr != nil {
			result.Errors = append(result.Errors, entry.Path+": "+readErr.Error())
			continue
		}
		content := ""
		if read != nil {
			content = read.Content
		}
		if len(content) > maxConfigFileBytes {
			result.Errors = append(result.Errors, entry.Path+": file too large")
			continue
		}
		if writeErr := launcherSvc.WriteInstanceFile(ctx, owner, instanceID, dest, content); writeErr != nil {
			result.Errors = append(result.Errors, dest+": "+writeErr.Error())
			continue
		}
		result.ClientConfigsInstalled = append(result.ClientConfigsInstalled, dest)
	}
}
