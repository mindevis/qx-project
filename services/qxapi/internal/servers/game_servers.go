package servers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type pendingProvision struct {
	phase        string
	vpsServerID  string
	gameServerID string
}

type GameServerView struct {
	ID                    string    `json:"id"`
	Name                  string    `json:"name"`
	ServerType            string    `json:"server_type"`
	MCVersion             string    `json:"mc_version"`
	LoaderVersion         *string   `json:"loader_version,omitempty"`
	Address               *string   `json:"address,omitempty"`
	Port                  int       `json:"port"`
	RconPassword          *string   `json:"rcon_password,omitempty"`
	RconPort              int       `json:"rcon_port"`
	Status                string    `json:"status"`
	ShowInMonitoring      bool      `json:"show_in_monitoring"`
	MonitoringDescription string    `json:"monitoring_description,omitempty"`
	BannerURL             string    `json:"banner_url,omitempty"`
	MonitoringTags        []string  `json:"monitoring_tags"`
	LastError             string    `json:"last_error,omitempty"`
	MinMemoryMB           *int      `json:"min_memory_mb,omitempty"`
	MaxMemoryMB           *int      `json:"max_memory_mb,omitempty"`
	ExtraJVMArgs          []string  `json:"extra_jvm_args,omitempty"`
	ExtraArgs             []string  `json:"extra_args,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type CreateGameServerInput struct {
	Name                  string
	ServerType            string
	MCVersion             string
	LoaderVersion         string
	Address               string
	Port                  int
	ShowInMonitoring      bool
	MonitoringDescription string
	BannerURL             string
	MonitoringTags        []string
}

type UpdateGameServerInput struct {
	Name                  *string
	Address               *string
	Port                  *int
	ShowInMonitoring      *bool
	MonitoringDescription *string
	BannerURL             *string
	MonitoringTags        []string
	MinMemoryMB           *int
	MaxMemoryMB           *int
	ExtraJVMArgs          *[]string
	ExtraArgs             *[]string
}

type ChangeGameServerVersionInput struct {
	MCVersion     string
	LoaderVersion string
}

const (
	gameServerProvisionTimeout = 25 * time.Minute
	gameServerRemoveTimeout    = 2 * time.Minute
	gameServerCopyTimeout      = 15 * time.Minute
)

func (s *Service) ListGameServers(ctx context.Context, ownerID, vpsID string) ([]GameServerView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	s.expireStaleGameServerProvisions(ctx, vpsID)
	var items []models.GameServer
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	out := make([]GameServerView, 0, len(items))
	for _, item := range items {
		out = append(out, gameServerViewFromModel(&item))
	}
	return out, nil
}

func (s *Service) CreateGameServer(ctx context.Context, ownerID, vpsID string, in CreateGameServerInput) (*GameServerView, error) {
	server, err := s.getOwned(ctx, ownerID, vpsID)
	if err != nil {
		return nil, err
	}
	if server.AgentTokenHash == nil {
		return nil, ErrNotDeployed
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return nil, ErrAgentOffline
	}

	name := strings.TrimSpace(in.Name)
	serverType := strings.TrimSpace(in.ServerType)
	mcVersion := strings.TrimSpace(in.MCVersion)
	loaderVersion := strings.TrimSpace(in.LoaderVersion)
	if name == "" || serverType == "" || mcVersion == "" {
		return nil, ErrValidation
	}
	if serverType != models.ServerTypeVanilla && loaderVersion == "" {
		return nil, ErrValidation
	}
	port := in.Port
	if port <= 0 {
		port = 25565
	}

	now := time.Now().UTC()
	gameServerID := uuid.NewString()
	workDir := fmt.Sprintf("/opt/qxsystem/server/instances/%s", gameServerID)
	var address *string
	if addr := strings.TrimSpace(in.Address); addr != "" {
		address = &addr
	}
	var loaderPtr *string
	if loaderVersion != "" {
		loaderPtr = &loaderVersion
	}
	rconPassword, err := generateRconPassword()
	if err != nil {
		return nil, err
	}

	item := models.GameServer{
		ID:                    gameServerID,
		ServerID:              vpsID,
		Name:                  name,
		ServerType:            serverType,
		MCVersion:             mcVersion,
		LoaderVersion:         loaderPtr,
		Address:               address,
		Port:                  port,
		RconPassword:          &rconPassword,
		Status:                models.GameServerStatusInstalling,
		WorkDir:               workDir,
		ShowInMonitoring:      in.ShowInMonitoring,
		MonitoringDescription: strings.TrimSpace(in.MonitoringDescription),
		BannerURL:             strings.TrimSpace(in.BannerURL),
		MonitoringTagsJSON:    encodeStringListJSON(in.MonitoringTags),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	minRAM := defaultGameServerMemoryMB
	maxRAM := defaultGameServerMemoryMB
	item.MinMemoryMB = &minRAM
	item.MaxMemoryMB = &maxRAM
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, err
	}

	if err := s.beginGameServerInstall(ctx, vpsID, &item); err != nil {
		_ = s.db.WithContext(ctx).Model(&item).Updates(map[string]any{
			"status":     models.GameServerStatusError,
			"updated_at": time.Now().UTC(),
		}).Error
		return nil, err
	}

	view := gameServerViewFromModel(&item)
	return &view, nil
}

func cloneGameServerDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Server"
	}
	suffix := " (copy)"
	if len(name)+len(suffix) > 128 {
		name = strings.TrimSpace(name[:128-len(suffix)])
	}
	return name + suffix
}

func remapWorkDirValue(value, srcWorkDir, destWorkDir string) string {
	if value == "" || srcWorkDir == "" || destWorkDir == "" {
		return value
	}
	return strings.ReplaceAll(value, srcWorkDir, destWorkDir)
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := *v
	return &s
}

func (s *Service) nextFreeGameServerPort(ctx context.Context, vpsID string) (int, error) {
	var items []models.GameServer
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Find(&items).Error; err != nil {
		return 0, err
	}
	used := make(map[int]struct{}, len(items)*2)
	for _, item := range items {
		if item.Port > 0 {
			used[item.Port] = struct{}{}
			used[rconPortFor(item.Port)] = struct{}{}
		}
	}
	for port := 25565; port < 55535; port++ {
		if _, ok := used[port]; ok {
			continue
		}
		if _, ok := used[rconPortFor(port)]; ok {
			continue
		}
		return port, nil
	}
	return 0, ErrValidation
}

func (s *Service) CloneGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*GameServerView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	var src models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&src).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if src.Status == models.GameServerStatusInstalling || src.Status == models.GameServerStatusStarting {
		return nil, ErrGameServerBusy
	}
	if strings.TrimSpace(src.WorkDir) == "" {
		return nil, ErrGameServerNotInstalled
	}

	port, err := s.nextFreeGameServerPort(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	rconPassword, err := generateRconPassword()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	destID := uuid.NewString()
	destWorkDir := fmt.Sprintf("/opt/qxsystem/server/instances/%s", destID)
	dest := models.GameServer{
		ID:                    destID,
		ServerID:              vpsID,
		Name:                  cloneGameServerDisplayName(src.Name),
		ServerType:            src.ServerType,
		MCVersion:             src.MCVersion,
		LoaderVersion:         cloneStringPtr(src.LoaderVersion),
		Address:               cloneStringPtr(src.Address),
		Port:                  port,
		RconPassword:          &rconPassword,
		Status:                models.GameServerStatusInstalling,
		WorkDir:               destWorkDir,
		StartCommand:          remapWorkDirValue(src.StartCommand, src.WorkDir, destWorkDir),
		StartArgsJSON:         remapWorkDirValue(src.StartArgsJSON, src.WorkDir, destWorkDir),
		JarPath:               remapWorkDirValue(src.JarPath, src.WorkDir, destWorkDir),
		ShowInMonitoring:      false,
		MonitoringDescription: src.MonitoringDescription,
		BannerURL:             src.BannerURL,
		MonitoringTagsJSON:    src.MonitoringTagsJSON,
		ContentResources:      src.ContentResources,
		MinMemoryMB:           cloneIntPtr(src.MinMemoryMB),
		MaxMemoryMB:           cloneIntPtr(src.MaxMemoryMB),
		ExtraJVMArgs:          append(models.StringList{}, src.ExtraJVMArgs...),
		ExtraArgs:             append(models.StringList{}, src.ExtraArgs...),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.db.WithContext(ctx).Create(&dest).Error; err != nil {
		return nil, err
	}
	if err := s.copyGameServerWorkDir(ctx, vpsID, &src, &dest); err != nil {
		_ = s.db.WithContext(context.Background()).Where("id = ?", dest.ID).Delete(&models.GameServer{}).Error
		_ = s.removeGameServerWorkDir(context.Background(), vpsID, &dest)
		return nil, err
	}
	dest.Status = models.GameServerStatusStopped
	if err := s.db.WithContext(ctx).Model(&dest).Updates(map[string]any{
		"status":     dest.Status,
		"updated_at": time.Now().UTC(),
	}).Error; err != nil {
		return nil, err
	}
	s.syncGameServerProperties(ctx, vpsID, &dest)
	view := gameServerViewFromModel(&dest)
	return &view, nil
}

func (s *Service) copyGameServerWorkDir(ctx context.Context, vpsID string, src, dest *models.GameServer) error {
	payload, err := json.Marshal(protocol.GameServerCopyPayload{
		GameServerID:  dest.ID,
		SourceWorkDir: src.WorkDir,
		DestWorkDir:   dest.WorkDir,
	})
	if err != nil {
		return err
	}
	raw, err := s.agentRPCWait(ctx, vpsID, protocol.TypeCmdServerCopy, protocol.TypeResServerCopy, payload, gameServerCopyTimeout)
	if err != nil {
		return err
	}
	var res struct {
		Error  string `json:"error"`
		Status string `json:"status"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &res)
	}
	if strings.TrimSpace(res.Error) != "" {
		return fmt.Errorf("copy work dir: %s", res.Error)
	}
	return nil
}

func normalizeGameServerVersions(serverType, mcVersion, loaderVersion string) (string, string, error) {
	mcVersion = strings.TrimSpace(mcVersion)
	loaderVersion = strings.TrimSpace(loaderVersion)
	if mcVersion == "" {
		return "", "", ErrValidation
	}
	if serverType != models.ServerTypeVanilla && loaderVersion == "" {
		return "", "", ErrValidation
	}
	if serverType == models.ServerTypeVanilla {
		loaderVersion = ""
	}
	return mcVersion, loaderVersion, nil
}

func (s *Service) ChangeGameServerVersion(ctx context.Context, ownerID, vpsID, gameServerID string, in ChangeGameServerVersionInput) (*GameServerView, error) {
	server, err := s.getOwned(ctx, ownerID, vpsID)
	if err != nil {
		return nil, err
	}
	if server.AgentTokenHash == nil {
		return nil, ErrNotDeployed
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return nil, ErrAgentOffline
	}

	var item models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if item.Status == models.GameServerStatusInstalling || item.Status == models.GameServerStatusStarting {
		return nil, ErrGameServerBusy
	}

	mcVersion, loaderVersion, err := normalizeGameServerVersions(item.ServerType, in.MCVersion, in.LoaderVersion)
	if err != nil {
		return nil, err
	}
	currentLoader := ""
	if item.LoaderVersion != nil {
		currentLoader = strings.TrimSpace(*item.LoaderVersion)
	}
	if item.MCVersion == mcVersion && currentLoader == loaderVersion {
		view := gameServerViewFromModel(&item)
		return &view, nil
	}

	wasRunning := item.Status == models.GameServerStatusRunning
	now := time.Now().UTC()
	updates := map[string]any{
		"mc_version": mcVersion,
		"status":     models.GameServerStatusInstalling,
		"updated_at": now,
	}
	if loaderVersion == "" {
		updates["loader_version"] = nil
	} else {
		updates["loader_version"] = loaderVersion
	}
	if err := s.db.WithContext(ctx).Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}
	item.MCVersion = mcVersion
	if loaderVersion == "" {
		item.LoaderVersion = nil
	} else {
		item.LoaderVersion = &loaderVersion
	}
	item.Status = models.GameServerStatusInstalling
	item.UpdatedAt = now

	if wasRunning {
		if err := s.sendGameServerStop(ctx, vpsID, &item, "version"); err != nil {
			_ = s.db.WithContext(ctx).Model(&item).Updates(map[string]any{
				"status":     models.GameServerStatusError,
				"updated_at": time.Now().UTC(),
			}).Error
			return nil, err
		}
	} else if err := s.beginGameServerInstall(ctx, vpsID, &item); err != nil {
		_ = s.db.WithContext(ctx).Model(&item).Updates(map[string]any{
			"status":     models.GameServerStatusError,
			"updated_at": time.Now().UTC(),
		}).Error
		return nil, err
	}

	view := gameServerViewFromModel(&item)
	return &view, nil
}

func (s *Service) ReinstallGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*GameServerView, error) {
	server, err := s.getOwned(ctx, ownerID, vpsID)
	if err != nil {
		return nil, err
	}
	if server.AgentTokenHash == nil {
		return nil, ErrNotDeployed
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return nil, ErrAgentOffline
	}

	var item models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if item.Status == models.GameServerStatusInstalling || item.Status == models.GameServerStatusStarting {
		return nil, ErrGameServerBusy
	}

	wasRunning := item.Status == models.GameServerStatusRunning || item.Status == models.GameServerStatusStarting

	rconPassword, err := generateRconPassword()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&item).Updates(map[string]any{
		"status":        models.GameServerStatusInstalling,
		"rcon_password": rconPassword,
		"updated_at":    now,
	}).Error; err != nil {
		return nil, err
	}
	item.Status = models.GameServerStatusInstalling
	item.RconPassword = &rconPassword
	item.UpdatedAt = now

	if wasRunning {
		if err := s.sendGameServerStop(ctx, vpsID, &item, "reinstall"); err != nil {
			_ = s.db.WithContext(ctx).Model(&item).Updates(map[string]any{
				"status":     models.GameServerStatusError,
				"updated_at": time.Now().UTC(),
			}).Error
			return nil, err
		}
	} else if err := s.wipeAndBeginGameServerInstall(ctx, vpsID, &item); err != nil {
		_ = s.db.WithContext(ctx).Model(&item).Updates(map[string]any{
			"status":     models.GameServerStatusError,
			"updated_at": time.Now().UTC(),
		}).Error
		return nil, err
	}

	view := gameServerViewFromModel(&item)
	return &view, nil
}

func (s *Service) StartGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*GameServerView, error) {
	item, err := s.getOwnedGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	if item.Status == models.GameServerStatusInstalling || item.Status == models.GameServerStatusStarting {
		return nil, ErrGameServerBusy
	}
	if item.Status == models.GameServerStatusRunning {
		return nil, ErrGameServerAlreadyRunning
	}
	if strings.TrimSpace(item.StartCommand) == "" && strings.TrimSpace(item.JarPath) == "" {
		return nil, ErrGameServerNotInstalled
	}
	if err := s.startGameServerProcess(ctx, vpsID, item); err != nil {
		return nil, err
	}
	view := gameServerViewFromModel(item)
	view.Status = models.GameServerStatusStarting
	return &view, nil
}

func (s *Service) StopGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*GameServerView, error) {
	item, err := s.getOwnedGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	if item.Status != models.GameServerStatusRunning && item.Status != models.GameServerStatusStarting {
		return nil, ErrGameServerNotRunning
	}
	if err := s.sendGameServerStop(ctx, vpsID, item, "stop"); err != nil {
		return nil, err
	}
	view := gameServerViewFromModel(item)
	return &view, nil
}

func (s *Service) RestartGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*GameServerView, error) {
	item, err := s.getOwnedGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	if item.Status == models.GameServerStatusInstalling || item.Status == models.GameServerStatusStarting {
		return nil, ErrGameServerBusy
	}
	if strings.TrimSpace(item.StartCommand) == "" && strings.TrimSpace(item.JarPath) == "" {
		return nil, ErrGameServerNotInstalled
	}
	if item.Status == models.GameServerStatusRunning || item.Status == models.GameServerStatusStarting {
		if err := s.sendGameServerStop(ctx, vpsID, item, "restart"); err != nil {
			return nil, err
		}
		view := gameServerViewFromModel(item)
		return &view, nil
	}
	if err := s.startGameServerProcess(ctx, vpsID, item); err != nil {
		return nil, err
	}
	view := gameServerViewFromModel(item)
	view.Status = models.GameServerStatusStarting
	return &view, nil
}

func (s *Service) getOwnedGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*models.GameServer, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	var item models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (s *Service) requireAgentOnline(ctx context.Context, vpsID string) error {
	server, err := s.getByID(ctx, vpsID)
	if err != nil {
		return err
	}
	if server.AgentTokenHash == nil {
		return ErrNotDeployed
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return ErrAgentOffline
	}
	return nil
}

func (s *Service) startGameServerProcess(ctx context.Context, vpsID string, item *models.GameServer) error {
	args, jvmArgs, extraArgs := gameServerStartArgSets(item)
	javaBin := ""
	server, err := s.getByID(ctx, vpsID)
	if err == nil {
		if cfg, err := parseConfig(server.ConfigJSON); err == nil {
			javaBin = cfg.JavaBin
			if len(args) == 0 && strings.TrimSpace(item.StartCommand) != "" {
				args = cfg.Args
			}
		}
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(item).Updates(map[string]any{
		"status":     models.GameServerStatusStarting,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}
	item.Status = models.GameServerStatusStarting
	return s.beginGameServerStart(ctx, vpsID, item.ID, item.JarPath, item.WorkDir, item.StartCommand, args, jvmArgs, extraArgs, javaBin, item.ServerType, item.MCVersion)
}

func (s *Service) sendGameServerStop(ctx context.Context, vpsID string, item *models.GameServer, phase string) error {
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return err
	}
	requestID := uuid.NewString()
	s.pending.Store(requestID, pendingProvision{
		phase:        phase,
		vpsServerID:  vpsID,
		gameServerID: item.ID,
	})
	payload, _ := json.Marshal(protocol.ServerStopPayload{
		Graceful:     true,
		TimeoutSec:   30,
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
	})
	return s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		Type:      protocol.TypeCmdServerStop,
		RequestID: requestID,
		Payload:   payload,
	})
}

func (s *Service) DeleteGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) error {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return err
	}
	var item models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if err := s.removeGameServerWorkDir(ctx, vpsID, &item); err != nil {
		return err
	}
	s.forgetActiveGameServer(ctx, vpsID, gameServerID)
	s.cleanupGameServerRecords(ctx, gameServerID)
	res := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).Delete(&models.GameServer{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) UpdateGameServer(ctx context.Context, ownerID, vpsID, gameServerID string, in UpdateGameServerInput) (*GameServerView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}

	var item models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, vpsID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if item.Status == models.GameServerStatusInstalling || item.Status == models.GameServerStatusStarting {
		return nil, ErrGameServerBusy
	}
	if in.Name == nil && in.Address == nil && in.Port == nil && !in.hasMonitoringUpdate() && !in.hasLaunchUpdate() {
		return nil, ErrValidation
	}

	updates := map[string]any{
		"updated_at": time.Now().UTC(),
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrValidation
		}
		updates["name"] = name
		item.Name = name
	}
	if in.Address != nil {
		addr := strings.TrimSpace(*in.Address)
		var address *string
		if addr != "" {
			address = &addr
		}
		updates["address"] = address
		item.Address = address
	}
	if in.Port != nil {
		port := *in.Port
		if port <= 0 || port > 65535 {
			return nil, ErrValidation
		}
		updates["port"] = port
		item.Port = port
	}
	if in.ShowInMonitoring != nil {
		updates["show_in_monitoring"] = *in.ShowInMonitoring
		item.ShowInMonitoring = *in.ShowInMonitoring
	}
	if in.MonitoringDescription != nil {
		desc := strings.TrimSpace(*in.MonitoringDescription)
		updates["monitoring_description"] = desc
		item.MonitoringDescription = desc
	}
	if in.BannerURL != nil {
		banner := strings.TrimSpace(*in.BannerURL)
		updates["banner_url"] = banner
		item.BannerURL = banner
	}
	if in.MonitoringTags != nil {
		tagsJSON := encodeStringListJSON(in.MonitoringTags)
		updates["monitoring_tags_json"] = tagsJSON
		item.MonitoringTagsJSON = tagsJSON
	}
	if in.MinMemoryMB != nil {
		if err := validateGameServerMemoryMB(*in.MinMemoryMB); err != nil {
			return nil, err
		}
		updates["min_memory_mb"] = *in.MinMemoryMB
		item.MinMemoryMB = in.MinMemoryMB
	}
	if in.MaxMemoryMB != nil {
		if err := validateGameServerMemoryMB(*in.MaxMemoryMB); err != nil {
			return nil, err
		}
		updates["max_memory_mb"] = *in.MaxMemoryMB
		item.MaxMemoryMB = in.MaxMemoryMB
	}
	if item.MinMemoryMB != nil && item.MaxMemoryMB != nil && *item.MinMemoryMB > *item.MaxMemoryMB {
		return nil, ErrValidation
	}
	if in.ExtraJVMArgs != nil {
		clean, err := sanitizeGameServerExecArgs(*in.ExtraJVMArgs)
		if err != nil {
			return nil, err
		}
		item.ExtraJVMArgs = models.StringList(clean)
		updates["extra_jvm_args"] = item.ExtraJVMArgs
	}
	if in.ExtraArgs != nil {
		clean, err := sanitizeGameServerExecArgs(*in.ExtraArgs)
		if err != nil {
			return nil, err
		}
		item.ExtraArgs = models.StringList(clean)
		updates["extra_args"] = item.ExtraArgs
	}

	if err := s.db.WithContext(ctx).Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.syncGameServerProperties(ctx, vpsID, &item)
	if item.ShowInMonitoring {
		s.refreshMonitoringSnapshots(ctx, ownerID, vpsID, &item)
	}

	view := gameServerViewFromModel(&item)
	return &view, nil
}

func (s *Service) wipeAndBeginGameServerInstall(ctx context.Context, vpsID string, item *models.GameServer) error {
	if err := s.wipeGameServerWorkDir(ctx, vpsID, item); err != nil {
		return err
	}
	if err := s.clearGameServerContentResources(ctx, item); err != nil {
		return err
	}
	return s.beginGameServerInstall(ctx, vpsID, item)
}

func (s *Service) clearGameServerContentResources(ctx context.Context, item *models.GameServer) error {
	if item == nil {
		return ErrValidation
	}
	item.ContentResources = models.InstanceResourceList{}
	return s.db.WithContext(ctx).Model(item).Update("content_resources", item.ContentResources).Error
}

func (s *Service) wipeGameServerWorkDir(ctx context.Context, vpsID string, item *models.GameServer) error {
	return s.sendGameServerWorkDirCommand(ctx, vpsID, item, false, agentRPCTimeout)
}

func (s *Service) removeGameServerWorkDir(ctx context.Context, vpsID string, item *models.GameServer) error {
	if item == nil || strings.TrimSpace(item.WorkDir) == "" {
		return nil
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return err
	}
	return s.sendGameServerWorkDirCommand(ctx, vpsID, item, true, gameServerRemoveTimeout)
}

func (s *Service) sendGameServerWorkDirCommand(ctx context.Context, vpsID string, item *models.GameServer, remove bool, timeout time.Duration) error {
	if item == nil || strings.TrimSpace(item.WorkDir) == "" {
		return ErrValidation
	}
	payload, err := json.Marshal(protocol.GameServerWorkDirPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Remove:       remove,
	})
	if err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = agentRPCTimeout
	}
	_, err = s.agentRPCWait(ctx, vpsID, protocol.TypeCmdServerWipe, protocol.TypeResServerWipe, payload, timeout)
	return err
}

func (s *Service) beginGameServerInstall(ctx context.Context, vpsID string, item *models.GameServer) error {
	loaderVersion := ""
	if item.LoaderVersion != nil {
		loaderVersion = *item.LoaderVersion
	}
	address := ""
	if item.Address != nil {
		address = *item.Address
	}
	rconPassword := ""
	if item.RconPassword != nil {
		rconPassword = *item.RconPassword
	}
	payload, err := json.Marshal(protocol.ServerInstallPayload{
		GameServerID:  item.ID,
		ServerType:    item.ServerType,
		MCVersion:     item.MCVersion,
		LoaderVersion: loaderVersion,
		WorkDir:       item.WorkDir,
		Name:          item.Name,
		Address:       address,
		Port:          item.Port,
		RconPassword:  rconPassword,
	})
	if err != nil {
		return err
	}
	requestID := uuid.NewString()
	s.pending.Store(requestID, pendingProvision{
		phase:        "install",
		vpsServerID:  vpsID,
		gameServerID: item.ID,
	})
	return s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		Type:      protocol.TypeCmdServerInstall,
		RequestID: requestID,
		Payload:   payload,
	})
}

func (s *Service) applyServerInstallResult(ctx context.Context, vpsID, requestID string, payload []byte) {
	raw, ok := s.pending.LoadAndDelete(requestID)
	if !ok {
		return
	}
	op := raw.(pendingProvision)
	if op.phase != "install" {
		return
	}

	var errPayload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(payload, &errPayload) == nil && strings.TrimSpace(errPayload.Error) != "" {
		s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError, "")
		if s.hub != nil {
			s.hub.BroadcastConsole(vpsID, protocol.ConsoleOutputPayload{
				Stream:       "stderr",
				Line:         "install failed: " + errPayload.Error,
				GameServerID: op.gameServerID,
			})
		}
		return
	}

	var result protocol.ServerInstallResult
	if err := json.Unmarshal(payload, &result); err != nil {
		s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError, "")
		return
	}

	argsJSON, _ := json.Marshal(result.Args)
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&models.GameServer{}).Where("id = ?", op.gameServerID).Updates(map[string]any{
		"work_dir":        result.WorkDir,
		"jar_path":        result.JarPath,
		"start_command":   result.Command,
		"start_args_json": string(argsJSON),
		"status":          models.GameServerStatusStarting,
		"updated_at":      now,
	}).Error; err != nil {
		return
	}

	var gameServer models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ?", op.gameServerID).First(&gameServer).Error; err != nil {
		return
	}
	args, jvmArgs, extraArgs := gameServerStartArgSets(&gameServer)
	_ = s.beginGameServerStart(ctx, vpsID, op.gameServerID, result.JarPath, result.WorkDir, result.Command, args, jvmArgs, extraArgs, result.JavaBin, gameServer.ServerType, gameServer.MCVersion)
	s.syncNetworksForGameServer(ctx, vpsID, op.gameServerID)
}

func (s *Service) beginGameServerStart(
	ctx context.Context,
	vpsID, gameServerID string,
	jarPath, workDir, command string,
	args, jvmArgs, extraArgs []string,
	javaBin, serverType, mcVersion string,
) error {
	now := time.Now().UTC()
	cfg := ServerConfig{
		JarPath:            jarPath,
		WorkDir:            workDir,
		Command:            command,
		Args:               args,
		JVMArgs:            jvmArgs,
		ExtraArgs:          extraArgs,
		JavaBin:            javaBin,
		ActiveGameServerID: gameServerID,
	}
	cfgBytes, _ := json.Marshal(cfg)
	updates := map[string]any{
		"server_type": serverType,
		"config_json": string(cfgBytes),
		"status":      models.ServerStatusStarting,
		"updated_at":  now,
	}
	if mcVersion != "" {
		updates["mc_version"] = mcVersion
	}
	_ = s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", vpsID).Updates(updates).Error

	startPayload, _ := json.Marshal(protocol.ServerStartPayload{
		GameServerID: gameServerID,
		ServerType:   serverType,
		JarPath:      jarPath,
		WorkDir:      workDir,
		Command:      command,
		Args:         args,
		JVMArgs:      jvmArgs,
		ExtraArgs:    extraArgs,
		JavaBin:      javaBin,
		MCVersion:    mcVersion,
	})
	startRequestID := uuid.NewString()
	s.pending.Store(startRequestID, pendingProvision{
		phase:        "start",
		vpsServerID:  vpsID,
		gameServerID: gameServerID,
	})
	return s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		Type:      protocol.TypeCmdServerStart,
		RequestID: startRequestID,
		Payload:   startPayload,
	})
}

func (s *Service) applyServerStopResult(ctx context.Context, serverID, requestID string, _ []byte) {
	_ = s.markMinecraftStopped(ctx, serverID)

	raw, ok := s.pending.LoadAndDelete(requestID)
	if !ok {
		return
	}
	op := raw.(pendingProvision)
	if op.gameServerID != "" {
		s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusStopped, "")
	}
	switch op.phase {
	case "restart":
		var item models.GameServer
		if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", op.gameServerID, serverID).First(&item).Error; err != nil {
			return
		}
		if err := s.startGameServerProcess(ctx, serverID, &item); err != nil {
			s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError, "")
		}
	case "reinstall":
		var item models.GameServer
		if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", op.gameServerID, serverID).First(&item).Error; err != nil {
			return
		}
		gameServerID := op.gameServerID
		vpsID := serverID
		go func() {
			bg := context.Background()
			if err := s.wipeAndBeginGameServerInstall(bg, vpsID, &item); err != nil {
				s.setGameServerStatus(bg, gameServerID, models.GameServerStatusError, "")
			}
		}()
	case "version":
		var item models.GameServer
		if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", op.gameServerID, serverID).First(&item).Error; err != nil {
			return
		}
		gameServerID := op.gameServerID
		vpsID := serverID
		go func() {
			bg := context.Background()
			now := time.Now().UTC()
			_ = s.db.WithContext(bg).Model(&item).Updates(map[string]any{
				"status":     models.GameServerStatusInstalling,
				"updated_at": now,
			}).Error
			item.Status = models.GameServerStatusInstalling
			item.UpdatedAt = now
			if err := s.beginGameServerInstall(bg, vpsID, &item); err != nil {
				s.setGameServerStatus(bg, gameServerID, models.GameServerStatusError, "")
			}
		}()
	}
}

func (s *Service) markPendingGameServerRunning(ctx context.Context, requestID string) {
	raw, ok := s.pending.LoadAndDelete(requestID)
	if !ok {
		return
	}
	op := raw.(pendingProvision)
	if op.phase != "start" {
		return
	}
	s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusRunning, "")
}

func (s *Service) markPendingGameServerError(ctx context.Context, requestID, message string) {
	raw, ok := s.pending.LoadAndDelete(requestID)
	if !ok {
		return
	}
	op := raw.(pendingProvision)
	if op.phase != "start" {
		return
	}
	s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError, message)
	if s.hub != nil && message != "" {
		s.hub.BroadcastConsole(op.vpsServerID, protocol.ConsoleOutputPayload{
			Stream:       "stderr",
			Line:         "start failed: " + message,
			GameServerID: op.gameServerID,
		})
	}
}

func (s *Service) setGameServerStatus(ctx context.Context, gameServerID, status, lastError string) {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if lastError != "" {
		updates["last_error"] = lastError
	} else if status == models.GameServerStatusRunning || status == models.GameServerStatusStopped {
		updates["last_error"] = ""
	}
	_ = s.db.WithContext(ctx).Model(&models.GameServer{}).Where("id = ?", gameServerID).Updates(updates).Error
}

func (s *Service) expireStaleGameServerProvisions(ctx context.Context, vpsID string) {
	cutoff := time.Now().UTC().Add(-gameServerProvisionTimeout)
	_ = s.db.WithContext(ctx).Model(&models.GameServer{}).
		Where(
			"server_id = ? AND status IN (?, ?) AND updated_at < ?",
			vpsID,
			models.GameServerStatusInstalling,
			models.GameServerStatusStarting,
			cutoff,
		).
		Update("status", models.GameServerStatusError).Error
}

func gameServerViewFromModel(item *models.GameServer) GameServerView {
	return GameServerView{
		ID:                    item.ID,
		Name:                  item.Name,
		ServerType:            item.ServerType,
		MCVersion:             item.MCVersion,
		LoaderVersion:         item.LoaderVersion,
		Address:               item.Address,
		Port:                  item.Port,
		RconPassword:          item.RconPassword,
		RconPort:              rconPortFor(item.Port),
		Status:                item.Status,
		ShowInMonitoring:      item.ShowInMonitoring,
		MonitoringDescription: strings.TrimSpace(item.MonitoringDescription),
		BannerURL:             strings.TrimSpace(item.BannerURL),
		MonitoringTags:        decodeStringListJSON(item.MonitoringTagsJSON),
		LastError:             strings.TrimSpace(item.LastError),
		MinMemoryMB:           item.MinMemoryMB,
		MaxMemoryMB:           item.MaxMemoryMB,
		ExtraJVMArgs:          append([]string{}, item.ExtraJVMArgs...),
		ExtraArgs:             append([]string{}, item.ExtraArgs...),
		CreatedAt:             item.CreatedAt,
	}
}

func (in UpdateGameServerInput) hasMonitoringUpdate() bool {
	return in.ShowInMonitoring != nil ||
		in.MonitoringDescription != nil ||
		in.BannerURL != nil ||
		in.MonitoringTags != nil
}

func (in UpdateGameServerInput) hasLaunchUpdate() bool {
	return in.MinMemoryMB != nil ||
		in.MaxMemoryMB != nil ||
		in.ExtraJVMArgs != nil ||
		in.ExtraArgs != nil
}

func gameServerIsInstalled(item *models.GameServer) bool {
	if item == nil || strings.TrimSpace(item.WorkDir) == "" {
		return false
	}
	return strings.TrimSpace(item.JarPath) != "" || strings.TrimSpace(item.StartCommand) != ""
}

func (s *Service) syncGameServerProperties(ctx context.Context, vpsID string, item *models.GameServer) {
	if !gameServerIsInstalled(item) {
		return
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return
	}
	rconPassword := ""
	if item.RconPassword != nil {
		rconPassword = strings.TrimSpace(*item.RconPassword)
	}
	if rconPassword == "" {
		return
	}
	address := ""
	if item.Address != nil {
		address = *item.Address
	}
	payload, err := json.Marshal(protocol.ServerConfigurePayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Name:         item.Name,
		Address:      address,
		Port:         item.Port,
		RconPassword: rconPassword,
	})
	if err != nil {
		return
	}
	_ = s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		Type:    protocol.TypeCmdServerConfigure,
		Payload: payload,
	})
}

// Ensure game server rows are removed when dedicated server is deleted.
func (s *Service) deleteGameServersForVPS(ctx context.Context, vpsID string) error {
	var items []models.GameServer
	if err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).Find(&items).Error; err != nil {
		return err
	}
	if s.hub != nil && s.hub.IsOnline(vpsID) {
		for i := range items {
			if err := s.removeGameServerWorkDir(ctx, vpsID, &items[i]); err != nil {
				return err
			}
		}
	}
	for _, item := range items {
		s.forgetActiveGameServer(ctx, vpsID, item.ID)
		s.cleanupGameServerRecords(ctx, item.ID)
	}
	return s.db.WithContext(ctx).Where("server_id = ?", vpsID).Delete(&models.GameServer{}).Error
}

func (s *Service) cleanupGameServerRecords(ctx context.Context, gameServerID string) {
	_ = s.db.WithContext(ctx).Where("game_server_id = ?", gameServerID).Delete(&models.GameServerInstanceBinding{}).Error
	_ = s.db.WithContext(ctx).Where("game_server_id = ?", gameServerID).Delete(&models.GameServerMonitoringFeedback{}).Error
	_ = s.db.WithContext(ctx).Model(&models.LauncherInstance{}).
		Where("managed_by_game_server_id = ?", gameServerID).
		Update("managed_by_game_server_id", nil).Error
}

func (s *Service) forgetActiveGameServer(ctx context.Context, vpsID, gameServerID string) {
	server, err := s.getByID(ctx, vpsID)
	if err != nil {
		return
	}
	cfg, err := parseConfig(server.ConfigJSON)
	if err != nil {
		return
	}
	if cfg.ActiveGameServerID != gameServerID {
		return
	}
	cfg.ActiveGameServerID = ""
	cfg.McPID = nil
	_ = s.saveConfig(ctx, vpsID, cfg)
}
