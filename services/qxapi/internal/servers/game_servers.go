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
}

const gameServerProvisionTimeout = 25 * time.Minute

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

	s.stopActiveGameServerProcess(ctx, vpsID, gameServerID)

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
	if err := s.sendGameServerStop(ctx, vpsID, gameServerID, "stop"); err != nil {
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
		if err := s.sendGameServerStop(ctx, vpsID, gameServerID, "restart"); err != nil {
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
	var args []string
	if item.StartArgsJSON != "" {
		_ = json.Unmarshal([]byte(item.StartArgsJSON), &args)
	}
	javaBin := ""
	server, err := s.getByID(ctx, vpsID)
	if err == nil {
		if cfg, err := parseConfig(server.ConfigJSON); err == nil {
			javaBin = cfg.JavaBin
			if len(args) == 0 {
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
	return s.beginGameServerStart(ctx, vpsID, item.ID, item.JarPath, item.WorkDir, item.StartCommand, args, nil, nil, javaBin, item.ServerType, item.MCVersion)
}

func (s *Service) sendGameServerStop(ctx context.Context, vpsID, gameServerID, phase string) error {
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return err
	}
	requestID := uuid.NewString()
	s.pending.Store(requestID, pendingProvision{
		phase:        phase,
		vpsServerID:  vpsID,
		gameServerID: gameServerID,
	})
	payload, _ := json.Marshal(protocol.ServerStopPayload{Graceful: true, TimeoutSec: 30})
	return s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		Type:      protocol.TypeCmdServerStop,
		RequestID: requestID,
		Payload:   payload,
	})
}

func (s *Service) stopActiveGameServerProcess(ctx context.Context, vpsID, gameServerID string) {
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
	if cfg.McPID == nil || *cfg.McPID <= 0 {
		return
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return
	}
	payload, _ := json.Marshal(protocol.ServerStopPayload{Graceful: true, TimeoutSec: 30})
	_ = s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		Type:    protocol.TypeCmdServerStop,
		Payload: payload,
	})
}

func (s *Service) DeleteGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) error {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return err
	}
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
	if in.Name == nil && in.Address == nil && in.Port == nil && !in.hasMonitoringUpdate() {
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
		s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError)
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
		s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError)
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
	_ = s.beginGameServerStart(ctx, vpsID, op.gameServerID, result.JarPath, result.WorkDir, result.Command, result.Args, result.JVMArgs, result.ExtraArgs, result.JavaBin, gameServer.ServerType, gameServer.MCVersion)
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
	if op.phase != "restart" {
		return
	}
	var item models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", op.gameServerID, serverID).First(&item).Error; err != nil {
		return
	}
	if err := s.startGameServerProcess(ctx, serverID, &item); err != nil {
		s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError)
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
	s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusRunning)
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
	s.setGameServerStatus(ctx, op.gameServerID, models.GameServerStatusError)
	if s.hub != nil && message != "" {
		s.hub.BroadcastConsole(op.vpsServerID, protocol.ConsoleOutputPayload{
			Stream:       "stderr",
			Line:         "start failed: " + message,
			GameServerID: op.gameServerID,
		})
	}
}

func (s *Service) setGameServerStatus(ctx context.Context, gameServerID, status string) {
	_ = s.db.WithContext(ctx).Model(&models.GameServer{}).Where("id = ?", gameServerID).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}).Error
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
		CreatedAt:             item.CreatedAt,
	}
}

func (in UpdateGameServerInput) hasMonitoringUpdate() bool {
	return in.ShowInMonitoring != nil ||
		in.MonitoringDescription != nil ||
		in.BannerURL != nil ||
		in.MonitoringTags != nil
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
	return s.db.WithContext(ctx).Where("server_id = ?", vpsID).Delete(&models.GameServer{}).Error
}
