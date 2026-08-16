package servers

import (
	"context"
	"strings"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

func contentResourceSide(modTarget string) string {
	switch strings.ToLower(strings.TrimSpace(modTarget)) {
	case "client-mods", "client-resourcepacks", "client-shaders":
		return "client"
	default:
		return "server"
	}
}

func appendUniqueContentResource(list models.InstanceResourceList, entry models.InstanceResourceEntry) models.InstanceResourceList {
	for i, existing := range list {
		if existing.Filename == entry.Filename && existing.ProjectID == entry.ProjectID && existing.ResourceType == entry.ResourceType {
			list[i] = entry
			return list
		}
	}
	return append(list, entry)
}

func removeContentResource(list models.InstanceResourceList, filename, resourceType, side string) models.InstanceResourceList {
	out := make(models.InstanceResourceList, 0, len(list))
	filename = strings.TrimSpace(filename)
	for _, entry := range list {
		if strings.EqualFold(entry.Filename, filename) &&
			(resourceType == "" || entry.ResourceType == resourceType) &&
			(side == "" || entry.SideOverride == side || (side == "server" && entry.SideOverride == "")) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func (s *Service) RecordGameServerResource(ctx context.Context, gameServerID string, entry models.InstanceResourceEntry) error {
	if entry.InstalledAt == "" {
		entry.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	}
	var gs models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ?", gameServerID).First(&gs).Error; err != nil {
		return err
	}
	gs.ContentResources = appendUniqueContentResource(gs.ContentResources, entry)
	return s.db.WithContext(ctx).Model(&gs).Update("content_resources", gs.ContentResources).Error
}

func (s *Service) RemoveGameServerResource(ctx context.Context, gameServerID, filename, resourceType, modTarget string) error {
	var gs models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ?", gameServerID).First(&gs).Error; err != nil {
		return err
	}
	gs.ContentResources = removeContentResource(gs.ContentResources, filename, resourceType, contentResourceSide(modTarget))
	return s.db.WithContext(ctx).Model(&gs).Update("content_resources", gs.ContentResources).Error
}

func (s *Service) ListGameServerResources(ctx context.Context, ownerID, vpsID, gameServerID, resourceType, modTarget string) ([]models.InstanceResourceEntry, error) {
	if _, err := s.GetGameServer(ctx, ownerID, vpsID, gameServerID); err != nil {
		return nil, err
	}
	var gs models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&gs).Error; err != nil {
		return nil, err
	}
	side := ""
	if strings.TrimSpace(modTarget) != "" {
		side = contentResourceSide(modTarget)
	}
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	out := make([]models.InstanceResourceEntry, 0, len(gs.ContentResources))
	for _, entry := range gs.ContentResources {
		if resourceType != "" && entry.ResourceType != resourceType {
			continue
		}
		entrySide := entry.SideOverride
		if entrySide == "" {
			entrySide = "server"
		}
		if side != "" && entrySide != side {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Service) RecordGameServerSync(ctx context.Context, gameServerID, kind string, body mods.SyncModRequest) error {
	return s.RecordGameServerResource(ctx, gameServerID, resourceEntryFromSync(kind, body))
}

func (s *Service) RecordGameServerUpload(ctx context.Context, gameServerID, kind, filename, modTarget string, size int64) error {
	return s.RecordGameServerResource(ctx, gameServerID, models.InstanceResourceEntry{
		Source:       "upload",
		ProjectName:  filename,
		Filename:     filename,
		ResourceType: kind,
		FileSize:     size,
		SideOverride: contentResourceSide(modTarget),
	})
}

func resourceEntryFromSync(kind string, body mods.SyncModRequest) models.InstanceResourceEntry {
	name := strings.TrimSpace(body.ProjectName)
	if name == "" {
		name = body.Filename
	}
	return models.InstanceResourceEntry{
		Source:        body.Source,
		ProjectID:     body.ProjectID,
		ProjectName:   name,
		VersionID:     body.VersionID,
		VersionNumber: body.VersionNumber,
		Filename:      body.Filename,
		ResourceType:  kind,
		IconURL:       body.IconURL,
		Downloads:     body.Downloads,
		FileSize:      body.FileSize,
		SideOverride:  contentResourceSide(body.ModTarget),
		InstalledAt:   time.Now().UTC().Format(time.RFC3339),
	}
}
