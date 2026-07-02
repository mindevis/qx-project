package servers

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type ConnectClientModEntry struct {
	Filename         string `json:"filename"`
	Size             int64  `json:"size,omitempty"`
	InstalledLocally bool   `json:"installed_locally"`
}

type ConnectModStatusView struct {
	ClientMods                       []ConnectClientModEntry `json:"client_mods"`
	AllClientModsInstalled           bool                    `json:"all_client_mods_installed"`
	SavedClientModEnabled            []string                `json:"saved_client_mod_enabled,omitempty"`
	ClientResourcepacks              []ConnectClientModEntry `json:"client_resourcepacks"`
	AllClientResourcepacksInstalled  bool                    `json:"all_client_resourcepacks_installed"`
	SavedClientResourcepackEnabled   []string                `json:"saved_client_resourcepack_enabled,omitempty"`
	ClientShaders                    []ConnectClientModEntry `json:"client_shaders"`
	AllClientShadersInstalled        bool                    `json:"all_client_shaders_installed"`
	SavedClientShaderEnabled         []string                `json:"saved_client_shader_enabled,omitempty"`
	ServerModCount                   int                     `json:"server_mod_count"`
	ServerResourcepackCount          int                     `json:"server_resourcepack_count"`
	ServerShaderCount                int                     `json:"server_shader_count"`
	AgentOnline                      bool                    `json:"agent_online"`
}

func (s *Service) GetConnectModStatus(
	ctx context.Context,
	userID, gameServerID, instanceID string,
	launcherSvc *launcher.Service,
) (*ConnectModStatusView, error) {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	instanceID = strings.TrimSpace(instanceID)
	if userID == "" || gameServerID == "" || instanceID == "" {
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

	localModFilenames := map[string]struct{}{}
	localResourcepackFilenames := map[string]struct{}{}
	localShaderFilenames := map[string]struct{}{}
	if launcherSvc != nil {
		owner := launcher.Owner{UserID: userID}
		resources, err := launcherSvc.ListInstanceResources(ctx, owner, instanceID)
		if err == nil {
			for _, res := range resources {
				if res.Filename == "" {
					continue
				}
				key := strings.ToLower(res.Filename)
				switch res.ResourceType {
				case "resourcepack":
					localResourcepackFilenames[key] = struct{}{}
				case "shader":
					localShaderFilenames[key] = struct{}{}
				default:
					if res.ResourceType == "mod" {
						localModFilenames[key] = struct{}{}
					}
				}
			}
		}
	}

	clientModNames := s.resolveClientModFilenames(ctx, userID, gs)
	clientResourcepackNames := s.resolveClientResourcepackFilenames(ctx, gs)
	clientShaderNames := s.resolveClientShaderFilenames(ctx, gs)
	serverModCount := len(s.resolveServerModFilenames(ctx, gs))
	serverResourcepackCount := len(s.resolveServerResourcepackFilenames(ctx, gs))
	serverShaderCount := len(s.resolveServerShaderFilenames(ctx, gs))

	clientMods, allModsInstalled := buildConnectClientFileEntries(clientModNames, localModFilenames)
	clientResourcepacks, allResourcepacksInstalled := buildConnectClientFileEntries(clientResourcepackNames, localResourcepackFilenames)
	clientShaders, allShadersInstalled := buildConnectClientFileEntries(clientShaderNames, localShaderFilenames)

	return &ConnectModStatusView{
		ClientMods:                      clientMods,
		AllClientModsInstalled:          allModsInstalled,
		SavedClientModEnabled:           []string(binding.ClientModEnabled),
		ClientResourcepacks:             clientResourcepacks,
		AllClientResourcepacksInstalled: allResourcepacksInstalled,
		SavedClientResourcepackEnabled:  []string(binding.ClientResourcepackEnabled),
		ClientShaders:                   clientShaders,
		AllClientShadersInstalled:       allShadersInstalled,
		SavedClientShaderEnabled:        []string(binding.ClientShaderEnabled),
		ServerModCount:                  serverModCount,
		ServerResourcepackCount:         serverResourcepackCount,
		ServerShaderCount:               serverShaderCount,
		AgentOnline:                     s.hub != nil && s.hub.IsOnline(gs.ServerID),
	}, nil
}

func (s *Service) resolveClientModFilenames(ctx context.Context, userID string, gs models.GameServer) []string {
	if s.hub != nil && s.hub.IsOnline(gs.ServerID) {
		var ownerID string
		var server models.Server
		if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err == nil {
			ownerID = server.OwnerID
		}
		if ownerID != "" {
			if entries, err := s.ListGameServerClientMods(ctx, ownerID, gs.ServerID, gs.ID); err == nil {
				return fileNamesFromEntries(entries)
			}
		}
	}
	return monitoringFilenameList(gs.MonitoringClientModsJSON)
}

func (s *Service) resolveServerModFilenames(ctx context.Context, gs models.GameServer) []string {
	if s.hub != nil && s.hub.IsOnline(gs.ServerID) {
		var ownerID string
		var server models.Server
		if err := s.db.WithContext(ctx).Where("id = ?", gs.ServerID).First(&server).Error; err == nil {
			ownerID = server.OwnerID
		}
		if ownerID != "" {
			if entries, err := s.ListGameServerMods(ctx, ownerID, gs.ServerID, gs.ID); err == nil {
				return fileNamesFromEntries(entries)
			}
		}
	}
	return monitoringFilenameList(gs.MonitoringModsJSON)
}

func (s *Service) SetClientModEnabled(ctx context.Context, userID, gameServerID string, enabled, resourcepacks, shaders []string) error {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	if userID == "" || gameServerID == "" {
		return ErrValidation
	}
	clean := cleanEnabledFilenames(enabled)
	cleanResourcepacks := cleanEnabledFilenames(resourcepacks)
	cleanShaders := cleanEnabledFilenames(shaders)
	res := s.db.WithContext(ctx).
		Model(&models.GameServerInstanceBinding{}).
		Where("user_id = ? AND game_server_id = ?", userID, gameServerID).
		Updates(map[string]any{
			"client_mod_enabled":          models.StringList(clean),
			"client_resourcepack_enabled": models.StringList(cleanResourcepacks),
			"client_shader_enabled":       models.StringList(cleanShaders),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func cleanEnabledFilenames(enabled []string) []string {
	clean := make([]string, 0, len(enabled))
	seen := map[string]struct{}{}
	for _, name := range enabled {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, name)
	}
	return clean
}
