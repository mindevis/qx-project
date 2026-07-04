package servers

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type PrepareConnectModsResult struct {
	ClientModsInstalled          []string `json:"client_mods_installed"`
	ServerModsInstalled          []string `json:"server_mods_installed"`
	ClientResourcepacksInstalled []string `json:"client_resourcepacks_installed"`
	ServerResourcepacksInstalled []string `json:"server_resourcepacks_installed"`
	ClientShadersInstalled       []string `json:"client_shaders_installed"`
	ServerShadersInstalled       []string `json:"server_shaders_installed"`
	Skipped                      []string `json:"skipped,omitempty"`
	Errors                       []string `json:"errors,omitempty"`
	AgentOnline                  bool     `json:"agent_online"`
}

func (s *Service) PrepareConnectMods(
	ctx context.Context,
	userID, gameServerID, instanceID string,
	launcherSvc *launcher.Service,
) (*PrepareConnectModsResult, error) {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	instanceID = strings.TrimSpace(instanceID)
	if userID == "" || gameServerID == "" || instanceID == "" {
		return nil, ErrValidation
	}
	if launcherSvc == nil {
		return nil, ErrValidation
	}

	var binding models.GameServerInstanceBinding
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND game_server_id = ? AND launcher_instance_id = ?", userID, gameServerID, instanceID).
		First(&binding).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var gs models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ?", gameServerID).First(&gs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := &PrepareConnectModsResult{
		ClientModsInstalled:          []string{},
		ServerModsInstalled:          []string{},
		ClientResourcepacksInstalled: []string{},
		ServerResourcepacksInstalled: []string{},
		ClientShadersInstalled:       []string{},
		ServerShadersInstalled:       []string{},
		AgentOnline:                  s.hub != nil && s.hub.IsOnline(gs.ServerID),
	}
	if !result.AgentOnline {
		return result, nil
	}

	ownerID, vpsID, err := s.gameServerOwnerAgent(ctx, gs)
	if err != nil {
		return result, nil
	}

	localByType := instanceResourceFilenamesByType(ctx, launcherSvc, userID, instanceID)
	owner := launcher.Owner{UserID: userID}

	pull := func(contentKind, modTarget, resourceType, filename string, installed *[]string) {
		key := strings.ToLower(filename)
		local := localByType[resourceType]
		if local == nil {
			local = map[string]struct{}{}
			localByType[resourceType] = local
		}
		if _, ok := local[key]; ok {
			result.Skipped = append(result.Skipped, filename)
			return
		}
		data, readErr := s.ReadGameServerContent(ctx, ownerID, vpsID, gs.ID, contentKind, modTarget, filename)
		if readErr != nil {
			result.Errors = append(result.Errors, filename+": "+readErr.Error())
			return
		}
		if _, uploadErr := launcherSvc.CreateInstanceResourceUpload(ctx, owner, instanceID, filename, resourceType, data); uploadErr != nil {
			result.Errors = append(result.Errors, filename+": "+uploadErr.Error())
			return
		}
		local[key] = struct{}{}
		*installed = append(*installed, filename)
	}

	enabledMods := enabledClientModSet(binding.ClientModEnabled)
	for _, name := range s.resolveClientModFilenames(ctx, userID, gs) {
		if !enabledMods[strings.ToLower(name)] {
			continue
		}
		pull("mod", "client-mods", "mod", name, &result.ClientModsInstalled)
	}

	// Server-only mods are intentionally not synced to launcher instances.
	enabledResourcepacks := enabledClientModSet(binding.ClientResourcepackEnabled)
	for _, name := range s.resolveClientResourcepackFilenames(ctx, gs) {
		if !enabledResourcepacks[strings.ToLower(name)] {
			continue
		}
		pull("resourcepack", "client-resourcepacks", "resourcepack", name, &result.ClientResourcepacksInstalled)
	}
	for _, name := range s.resolveServerResourcepackFilenames(ctx, gs) {
		pull("resourcepack", "", "resourcepack", name, &result.ServerResourcepacksInstalled)
	}

	enabledShaders := enabledClientModSet(binding.ClientShaderEnabled)
	for _, name := range s.resolveClientShaderFilenames(ctx, gs) {
		if !enabledShaders[strings.ToLower(name)] {
			continue
		}
		pull("shader", "client-shaders", "shader", name, &result.ClientShadersInstalled)
	}
	for _, name := range s.resolveServerShaderFilenames(ctx, gs) {
		pull("shader", "", "shader", name, &result.ServerShadersInstalled)
	}

	return result, nil
}

func (s *Service) gameServerOwnerAgent(ctx context.Context, gs models.GameServer) (ownerID, vpsID string, err error) {
	if s.hub == nil || !s.hub.IsOnline(gs.ServerID) {
		return "", "", ErrAgentOffline
	}
	var server models.Server
	if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err != nil {
		return "", "", err
	}
	if !gameServerIsInstalled(&gs) {
		return "", "", ErrGameServerNotInstalled
	}
	return server.OwnerID, gs.ServerID, nil
}

func instanceResourceFilenamesByType(ctx context.Context, launcherSvc *launcher.Service, userID, instanceID string) map[string]map[string]struct{} {
	out := map[string]map[string]struct{}{
		"mod":          {},
		"resourcepack": {},
		"shader":       {},
	}
	owner := launcher.Owner{UserID: userID}
	resources, err := launcherSvc.ListInstanceResources(ctx, owner, instanceID)
	if err != nil {
		return out
	}
	for _, res := range resources {
		if res.Filename == "" {
			continue
		}
		resourceType := res.ResourceType
		if resourceType == "" {
			resourceType = "mod"
		}
		if out[resourceType] == nil {
			out[resourceType] = map[string]struct{}{}
		}
		out[resourceType][strings.ToLower(res.Filename)] = struct{}{}
	}
	return out
}

func enabledClientModSet(enabled []string) map[string]bool {
	out := map[string]bool{}
	for _, name := range enabled {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out[strings.ToLower(name)] = true
	}
	return out
}
