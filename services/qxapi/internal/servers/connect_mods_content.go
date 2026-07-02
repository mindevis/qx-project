package servers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func (s *Service) resolveClientResourcepackFilenames(ctx context.Context, gs models.GameServer) []string {
	if s.hub != nil && s.hub.IsOnline(gs.ServerID) {
		var ownerID string
		var server models.Server
		if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err == nil {
			ownerID = server.OwnerID
		}
		if ownerID != "" {
			if entries, err := s.ListGameServerClientResourcepacks(ctx, ownerID, gs.ServerID, gs.ID); err == nil {
				return fileNamesFromEntries(entries)
			}
		}
	}
	return monitoringFilenameList(gs.MonitoringClientResourcepacksJSON)
}

func (s *Service) resolveClientShaderFilenames(ctx context.Context, gs models.GameServer) []string {
	if s.hub != nil && s.hub.IsOnline(gs.ServerID) {
		var ownerID string
		var server models.Server
		if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err == nil {
			ownerID = server.OwnerID
		}
		if ownerID != "" {
			if entries, err := s.ListGameServerClientShaders(ctx, ownerID, gs.ServerID, gs.ID); err == nil {
				return fileNamesFromEntries(entries)
			}
		}
	}
	return monitoringFilenameList(gs.MonitoringClientShadersJSON)
}

func (s *Service) resolveServerResourcepackFilenames(ctx context.Context, gs models.GameServer) []string {
	if s.hub != nil && s.hub.IsOnline(gs.ServerID) {
		var ownerID string
		var server models.Server
		if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err == nil {
			ownerID = server.OwnerID
		}
		if ownerID != "" {
			if entries, err := s.ListGameServerResourcepacks(ctx, ownerID, gs.ServerID, gs.ID); err == nil {
				return fileNamesFromEntries(entries)
			}
		}
	}
	return monitoringFilenameList(gs.MonitoringResourcepacksJSON)
}

func (s *Service) resolveServerShaderFilenames(ctx context.Context, gs models.GameServer) []string {
	if s.hub != nil && s.hub.IsOnline(gs.ServerID) {
		var ownerID string
		var server models.Server
		if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err == nil {
			ownerID = server.OwnerID
		}
		if ownerID != "" {
			if entries, err := s.ListGameServerShaders(ctx, ownerID, gs.ServerID, gs.ID); err == nil {
				return fileNamesFromEntries(entries)
			}
		}
	}
	return monitoringFilenameList(gs.MonitoringShadersJSON)
}

func fileNamesFromEntries(entries []protocol.FileEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.Dir {
			names = append(names, entry.Name)
		}
	}
	return names
}

func monitoringFilenameList(raw string) []string {
	if raw == "" {
		return nil
	}
	var names []string
	if json.Unmarshal([]byte(raw), &names) != nil {
		return nil
	}
	return names
}

func buildConnectClientFileEntries(names []string, localFilenames map[string]struct{}) ([]ConnectClientModEntry, bool) {
	clientFiles := make([]ConnectClientModEntry, 0, len(names))
	allInstalled := len(names) > 0
	for _, name := range names {
		installed := false
		if _, ok := localFilenames[strings.ToLower(name)]; ok {
			installed = true
		} else {
			allInstalled = false
		}
		clientFiles = append(clientFiles, ConnectClientModEntry{
			Filename:         name,
			InstalledLocally: installed,
		})
	}
	if len(names) == 0 {
		allInstalled = false
	}
	return clientFiles, allInstalled
}
