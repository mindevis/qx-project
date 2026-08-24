package servers

import (
	"context"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

func normalizeContentSide(side string) string {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "client", "server", "both":
		return strings.ToLower(strings.TrimSpace(side))
	default:
		return ""
	}
}

func contentResourceSide(modTarget, sideOverride string) string {
	if side := normalizeContentSide(sideOverride); side != "" {
		return side
	}
	switch strings.ToLower(strings.TrimSpace(modTarget)) {
	case "client-mods", "client-resourcepacks", "client-shaders":
		return "client"
	default:
		return "server"
	}
}

func effectiveContentSide(entry models.InstanceResourceEntry) string {
	if side := normalizeContentSide(entry.SideOverride); side != "" {
		return side
	}
	return "server"
}

func contentFolderKind(side string) string {
	if normalizeContentSide(side) == "client" {
		return "client"
	}
	return "server"
}

func contentModTargetForSide(kind, side string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "resourcepack":
		if contentFolderKind(side) == "client" {
			return "client-resourcepacks"
		}
		return ""
	case "shader":
		if contentFolderKind(side) == "client" {
			return "client-shaders"
		}
		return ""
	default:
		if contentFolderKind(side) == "client" {
			return "client-mods"
		}
		return ""
	}
}

func matchesContentTargetFilter(entrySide, folderSide string) bool {
	if folderSide == "" {
		return true
	}
	if folderSide == "client" {
		return entrySide == "client"
	}
	return entrySide == "server" || entrySide == "both"
}

func contentSideForFilename(resources models.InstanceResourceList, filename string) string {
	filename = strings.TrimSpace(filename)
	for _, entry := range resources {
		if !strings.EqualFold(entry.Filename, filename) {
			continue
		}
		if side := normalizeContentSide(entry.SideOverride); side != "" {
			return side
		}
		return "both"
	}
	return "both"
}

func shouldPullServerModToClient(side string) bool {
	return normalizeContentSide(side) != "server"
}

func contentResourceSameSlot(existing, entry models.InstanceResourceEntry) bool {
	if existing.ResourceType != "" && entry.ResourceType != "" && existing.ResourceType != entry.ResourceType {
		return false
	}
	if existing.ProjectID != "" && entry.ProjectID != "" &&
		strings.EqualFold(existing.Source, entry.Source) &&
		existing.ProjectID == entry.ProjectID {
		return contentFolderKind(effectiveContentSide(existing)) == contentFolderKind(effectiveContentSide(entry))
	}
	return strings.EqualFold(existing.Filename, entry.Filename)
}

func appendUniqueContentResource(list models.InstanceResourceList, entry models.InstanceResourceEntry) models.InstanceResourceList {
	for i, existing := range list {
		if contentResourceSameSlot(existing, entry) {
			list[i] = entry
			return list
		}
	}
	return append(list, entry)
}

func findContentResourceByProject(list models.InstanceResourceList, source, projectID, side string) *models.InstanceResourceEntry {
	source = strings.TrimSpace(source)
	projectID = strings.TrimSpace(projectID)
	if source == "" || projectID == "" {
		return nil
	}
	wantFolder := contentFolderKind(side)
	for i, entry := range list {
		if !strings.EqualFold(entry.Source, source) || entry.ProjectID != projectID {
			continue
		}
		if contentFolderKind(effectiveContentSide(entry)) != wantFolder {
			continue
		}
		found := list[i]
		return &found
	}
	return nil
}

func (s *Service) LookupGameServerContentResource(
	ctx context.Context,
	ownerID, vpsID, gameServerID, resourceType, source, projectID, modTarget, sideOverride string,
) (*models.InstanceResourceEntry, error) {
	items, err := s.ListGameServerResources(ctx, ownerID, vpsID, gameServerID, resourceType, "")
	if err != nil {
		return nil, err
	}
	return findContentResourceByProject(items, source, projectID, contentResourceSide(modTarget, sideOverride)), nil
}

func ShouldReplaceInstalledContent(existing *models.InstanceResourceEntry, replaceFilename, newFilename, newVersionID string) bool {
	if strings.TrimSpace(replaceFilename) != "" && !protocol.SameContentFilename(replaceFilename, newFilename) {
		return true
	}
	if existing == nil {
		return false
	}
	if existing.VersionID != "" && existing.VersionID == newVersionID && protocol.SameContentFilename(existing.Filename, newFilename) {
		return false
	}
	return existing.VersionID != newVersionID || !protocol.SameContentFilename(existing.Filename, newFilename)
}

func ContentFilesToReplace(existing *models.InstanceResourceEntry, replaceFilename, newFilename string) []string {
	names := make([]string, 0, 2)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || protocol.SameContentFilename(name, newFilename) {
			return
		}
		for _, have := range names {
			if protocol.SameContentFilename(have, name) {
				return
			}
		}
		names = append(names, name)
	}
	add(replaceFilename)
	if existing != nil {
		add(existing.Filename)
	}
	return names
}

func removeContentResource(list models.InstanceResourceList, filename, resourceType, side string) models.InstanceResourceList {
	out := make(models.InstanceResourceList, 0, len(list))
	filename = strings.TrimSpace(filename)
	for _, entry := range list {
		if strings.EqualFold(entry.Filename, filename) &&
			(resourceType == "" || entry.ResourceType == resourceType) &&
			matchesContentTargetFilter(effectiveContentSide(entry), side) {
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
	gs.ContentResources = removeContentResource(gs.ContentResources, filename, resourceType, contentResourceSide(modTarget, ""))
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
		side = contentResourceSide(modTarget, "")
	}
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	out := make([]models.InstanceResourceEntry, 0, len(gs.ContentResources))
	for _, entry := range gs.ContentResources {
		if resourceType != "" && entry.ResourceType != resourceType {
			continue
		}
		if !matchesContentTargetFilter(effectiveContentSide(entry), side) {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *Service) RecordGameServerSync(ctx context.Context, gameServerID, kind string, body mods.SyncModRequest) error {
	return s.RecordGameServerResource(ctx, gameServerID, resourceEntryFromSync(kind, body))
}

func (s *Service) RecordGameServerUpload(ctx context.Context, gameServerID, kind, filename, modTarget, sideOverride string, size int64) error {
	return s.RecordGameServerResource(ctx, gameServerID, models.InstanceResourceEntry{
		Source:       "upload",
		ProjectName:  filename,
		Filename:     filename,
		ResourceType: kind,
		FileSize:     size,
		SideOverride: contentResourceSide(modTarget, sideOverride),
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
		VersionType:   mods.InferVersionType(body.VersionType, body.VersionNumber, body.Filename),
		Filename:      body.Filename,
		ResourceType:  kind,
		IconURL:       body.IconURL,
		Downloads:     body.Downloads,
		FileSize:      body.FileSize,
		SideOverride:  contentResourceSide(body.ModTarget, body.SideOverride),
		InstalledAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *Service) UpdateGameServerResourceSide(
	ctx context.Context,
	ownerID, vpsID, gameServerID, filename, resourceType, side string,
) error {
	filename = strings.TrimSpace(filename)
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	side = normalizeContentSide(side)
	if filename == "" || side == "" {
		return ErrValidation
	}
	if resourceType == "" {
		resourceType = "mod"
	}
	if _, err := s.GetGameServer(ctx, ownerID, vpsID, gameServerID); err != nil {
		return err
	}
	var gs models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&gs).Error; err != nil {
		return err
	}

	idx := -1
	for i, entry := range gs.ContentResources {
		if strings.EqualFold(entry.Filename, filename) && (entry.ResourceType == resourceType || entry.ResourceType == "") {
			idx = i
			break
		}
	}

	currentFolder := contentFolderKind("server")
	if idx >= 0 {
		currentFolder = contentFolderKind(effectiveContentSide(gs.ContentResources[idx]))
	}
	if diskFolder, ok := s.inferContentFileFolder(ctx, ownerID, vpsID, gameServerID, resourceType, filename); ok {
		currentFolder = diskFolder
	}
	wantFolder := contentFolderKind(side)
	if currentFolder != wantFolder {
		fromTarget := contentModTargetForSide(resourceType, currentFolder)
		toTarget := contentModTargetForSide(resourceType, side)
		data, err := s.ReadGameServerContent(ctx, ownerID, vpsID, gameServerID, resourceType, fromTarget, filename)
		if err != nil {
			return err
		}
		if _, err := s.UploadGameServerContent(ctx, ownerID, vpsID, gameServerID, resourceType, toTarget, filename, data); err != nil {
			return err
		}
		if _, err := s.DeleteGameServerContent(ctx, ownerID, vpsID, gameServerID, resourceType, fromTarget, filename); err != nil {
			return err
		}
	}

	if idx >= 0 {
		gs.ContentResources[idx].SideOverride = side
		if gs.ContentResources[idx].ResourceType == "" {
			gs.ContentResources[idx].ResourceType = resourceType
		}
	} else {
		gs.ContentResources = append(gs.ContentResources, models.InstanceResourceEntry{
			Source:       "upload",
			ProjectName:  filename,
			Filename:     filename,
			ResourceType: resourceType,
			SideOverride: side,
			InstalledAt:  time.Now().UTC().Format(time.RFC3339),
		})
	}
	return s.db.WithContext(ctx).Model(&gs).Update("content_resources", gs.ContentResources).Error
}

func (s *Service) inferContentFileFolder(
	ctx context.Context,
	ownerID, vpsID, gameServerID, resourceType, filename string,
) (string, bool) {
	if strings.ToLower(strings.TrimSpace(resourceType)) != "mod" {
		return "", false
	}
	if entries, err := s.ListGameServerClientMods(ctx, ownerID, vpsID, gameServerID); err == nil {
		for _, entry := range entries {
			if !entry.Dir && strings.EqualFold(entry.Name, filename) {
				return "client", true
			}
		}
	}
	if entries, err := s.ListGameServerMods(ctx, ownerID, vpsID, gameServerID); err == nil {
		for _, entry := range entries {
			if !entry.Dir && strings.EqualFold(entry.Name, filename) {
				return "server", true
			}
		}
	}
	return "", false
}
