package servers

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrValidation    = errors.New("validation error")
	ErrForbidden     = errors.New("forbidden")
	ErrAgentOffline  = errors.New("agent offline")
	ErrNotDeployed   = errors.New("agent not deployed")
	ErrGameServerBusy          = errors.New("game server provisioning in progress")
	ErrGameServerNotInstalled  = errors.New("game server is not installed")
	ErrGameServerNotRunning    = errors.New("game server is not running")
	ErrGameServerAlreadyRunning = errors.New("game server already running")
)

const agentTokenTTL = 365 * 24 * time.Hour

type Service struct {
	db       *gorm.DB
	tokens   *auth.TokenService
	enc      *crypto.Encryptor
	hub      *agenthub.Hub
	deployer DeployExecutor
	pending    sync.Map
	pendingRPC sync.Map
}

type DeployExecutor interface {
	Deploy(ctx context.Context, serverID string, cred models.SSHCredential, agentToken string) error
}

type NoopDeployer struct{}

func (NoopDeployer) Deploy(context.Context, string, models.SSHCredential, string) error {
	return nil
}

type DeployOutput struct {
	View       *ServerView
	AgentToken string
}

func NewService(db *gorm.DB, tokens *auth.TokenService, enc *crypto.Encryptor, hub *agenthub.Hub, deployer DeployExecutor) *Service {
	if deployer == nil {
		deployer = NoopDeployer{}
	}
	return &Service{db: db, tokens: tokens, enc: enc, hub: hub, deployer: deployer}
}

func (s *Service) Hub() *agenthub.Hub {
	return s.hub
}

func (s *Service) OnAgentEvent(serverID string, env protocol.Envelope) {
	ctx := context.Background()
	now := time.Now().UTC()
	switch env.Type {
	case protocol.TypeEvtAgentHeartbeat:
		_ = s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("last_seen_at", now).Error
	case protocol.TypeResServerInstall:
		s.applyServerInstallResult(ctx, serverID, env.RequestID, env.Payload)
	case protocol.TypeResServerStart:
		s.applyServerStartResult(ctx, serverID, env.RequestID, env.Payload)
	case protocol.TypeResServerStop:
		s.applyServerStopResult(ctx, serverID, env.RequestID, env.Payload)
	case protocol.TypeEvtServerStatus:
		s.applyServerStatusEvent(ctx, serverID, env.Payload)
	default:
		if isRPCResponseType(env.Type) {
			s.deliverRPCResponse(env.RequestID, env.Payload)
		}
	}
	if env.Type == protocol.TypeEvtConsoleOutput {
		var payload protocol.ConsoleOutputPayload
		if err := json.Unmarshal(env.Payload, &payload); err == nil && s.hub != nil {
			s.hub.BroadcastConsole(serverID, payload)
		}
	}
}

type SSHInput struct {
	Host                  string
	Port                  int
	Username              string
	PrivateKey            string
	PrivateKeyPassphrase  string
}

type ServerConfig struct {
	JarPath            string   `json:"jar_path"`
	WorkDir            string   `json:"work_dir,omitempty"`
	Command            string   `json:"command,omitempty"`
	Args               []string `json:"args,omitempty"`
	JVMArgs            []string `json:"jvm_args"`
	ExtraArgs          []string `json:"extra_args"`
	JavaBin            string   `json:"java_bin,omitempty"`
	McPID              *int     `json:"mc_pid,omitempty"`
	ActiveGameServerID string   `json:"active_game_server_id,omitempty"`
}

type CreateServerInput struct {
	Name       string
	ServerType string
	MCVersion  string
	SSH        SSHInput
	Config     ServerConfig
}

type ServerView struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Slug             string        `json:"slug"`
	ServerType       string        `json:"server_type"`
	Status           string        `json:"status"`
	MCVersion        *string       `json:"mc_version,omitempty"`
	Config           ServerConfig  `json:"config"`
	SSH              SSHPublicView `json:"ssh"`
	AgentDeployed    bool          `json:"agent_deployed"`
	AgentOnline      bool          `json:"agent_online"`
	AgentVersion     *string       `json:"agent_version,omitempty"`
	MinecraftRunning bool          `json:"minecraft_running"`
	LastSeenAt       *time.Time    `json:"last_seen_at,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type SSHPublicView struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func (s *Service) List(ctx context.Context, ownerID string) ([]ServerView, error) {
	var items []models.Server
	if err := s.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	out := make([]ServerView, 0, len(items))
	for _, item := range items {
		view, err := s.viewFromModel(ctx, &item)
		if err != nil {
			return nil, err
		}
		out = append(out, *view)
	}
	return out, nil
}

func (s *Service) Create(ctx context.Context, ownerID string, in CreateServerInput) (*ServerView, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || strings.TrimSpace(in.SSH.Host) == "" || strings.TrimSpace(in.SSH.Username) == "" || strings.TrimSpace(in.SSH.PrivateKey) == "" {
		return nil, ErrValidation
	}
	serverType := strings.TrimSpace(in.ServerType)
	if serverType == "" {
		serverType = models.ServerTypeVanilla
	}
	port := in.SSH.Port
	if port <= 0 {
		port = 22
	}

	cfg := in.Config
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if len(cfgBytes) == 0 || string(cfgBytes) == "null" {
		cfgBytes = []byte("{}")
	}
	encKey, err := s.enc.Encrypt([]byte(in.SSH.PrivateKey))
	if err != nil {
		return nil, err
	}
	var passphraseEnc []byte
	if pass := strings.TrimSpace(in.SSH.PrivateKeyPassphrase); pass != "" {
		passphraseEnc, err = s.enc.Encrypt([]byte(pass))
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	slug := uniqueSlug(ctx, s.db, ownerID, slugify(name))
	server := models.Server{
		ID:         uuid.NewString(),
		OwnerID:    ownerID,
		Name:       name,
		Slug:       slug,
		ServerType: serverType,
		Status:     models.ServerStatusPending,
		ConfigJSON: string(cfgBytes),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if v := strings.TrimSpace(in.MCVersion); v != "" {
		server.MCVersion = &v
	}
	cred := models.SSHCredential{
		ServerID:                server.ID,
		Host:                    strings.TrimSpace(in.SSH.Host),
		Port:                    port,
		Username:                strings.TrimSpace(in.SSH.Username),
		PrivateKeyEnc:           encKey,
		PrivateKeyPassphraseEnc: passphraseEnc,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&server).Error; err != nil {
			return err
		}
		return tx.Create(&cred).Error
	}); err != nil {
		return nil, err
	}
	return s.viewFromModel(ctx, &server)
}

func (s *Service) Get(ctx context.Context, ownerID, serverID string) (*ServerView, error) {
	server, err := s.getOwned(ctx, ownerID, serverID)
	if err != nil {
		return nil, err
	}
	return s.viewFromModel(ctx, server)
}

func (s *Service) Delete(ctx context.Context, ownerID, serverID string) error {
	server, err := s.getOwned(ctx, ownerID, serverID)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Delete(server)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	_ = s.deleteGameServersForVPS(ctx, serverID)
	return nil
}

func (s *Service) Deploy(ctx context.Context, ownerID, serverID string) (*DeployOutput, error) {
	server, err := s.getOwned(ctx, ownerID, serverID)
	if err != nil {
		return nil, err
	}
	var cred models.SSHCredential
	if err := s.db.WithContext(ctx).Where("server_id = ?", serverID).First(&cred).Error; err != nil {
		return nil, err
	}

	token, err := s.tokens.IssueAgentToken(serverID, agentTokenTTL)
	if err != nil {
		return nil, err
	}
	hash := auth.HashToken(token)

	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(server).Updates(map[string]any{
		"status":           models.ServerStatusDeploying,
		"agent_token_hash": hash,
		"updated_at":       now,
	}).Error; err != nil {
		return nil, err
	}

	agent := models.Agent{
		ID:        uuid.NewString(),
		ServerID:  serverID,
		OS:        "linux",
		CreatedAt: now,
	}
	_ = s.db.WithContext(ctx).Where("server_id = ?", serverID).Delete(&models.Agent{}).Error
	if err := s.db.WithContext(ctx).Create(&agent).Error; err != nil {
		return nil, err
	}

	if err := s.deployer.Deploy(ctx, serverID, cred, token); err != nil {
		_ = s.db.WithContext(ctx).Model(server).Update("status", models.ServerStatusError).Error
		return nil, err
	}

	_ = s.db.WithContext(ctx).Model(server).Update("status", models.ServerStatusOffline).Error
	_ = s.setMcPID(ctx, serverID, nil)
	if err := s.db.WithContext(ctx).Where("id = ?", serverID).First(server).Error; err != nil {
		return nil, err
	}
	view, err := s.viewFromModel(ctx, server)
	if err != nil {
		return nil, err
	}
	return &DeployOutput{
		View:       view,
		AgentToken: token,
	}, nil
}

func (s *Service) ValidateAgentToken(ctx context.Context, serverID, token string) error {
	server, err := s.getByID(ctx, serverID)
	if err != nil {
		return err
	}
	if server.AgentTokenHash == nil || auth.HashToken(token) != *server.AgentTokenHash {
		return ErrForbidden
	}
	return nil
}

func (s *Service) applyServerStartResult(ctx context.Context, serverID, requestID string, payload []byte) {
	status := models.ServerStatusOffline
	var mcPID *int
	var errPayload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(payload, &errPayload) == nil && strings.TrimSpace(errPayload.Error) != "" {
		// Game server start failure is not a VPS host/SSH error; keep host offline.
		if s.hub != nil {
			s.hub.BroadcastConsole(serverID, protocol.ConsoleOutputPayload{
				Stream: "stderr",
				Line:   errPayload.Error,
			})
		}
		s.markPendingGameServerError(ctx, requestID, errPayload.Error)
	} else {
		var result protocol.ServerStartResult
		if json.Unmarshal(payload, &result) == nil && result.PID > 0 {
			status = models.ServerStatusOnline
			pid := result.PID
			mcPID = &pid
			s.markPendingGameServerRunning(ctx, requestID)
		}
	}
	_ = s.setMcPID(ctx, serverID, mcPID)
	_ = s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("status", status).Error
}

func (s *Service) markMinecraftStopped(ctx context.Context, serverID string) error {
	server, err := s.getByID(ctx, serverID)
	if err != nil {
		return err
	}
	cfg, err := parseConfig(server.ConfigJSON)
	if err != nil {
		return err
	}
	activeGameServerID := cfg.ActiveGameServerID
	if err := s.setMcPID(ctx, serverID, nil); err != nil {
		return err
	}
	if activeGameServerID != "" {
		s.setGameServerStatus(ctx, activeGameServerID, models.GameServerStatusStopped, "")
	}
	return s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("status", models.ServerStatusOffline).Error
}

func (s *Service) setMcPID(ctx context.Context, serverID string, pid *int) error {
	server, err := s.getByID(ctx, serverID)
	if err != nil {
		return err
	}
	cfg, err := parseConfig(server.ConfigJSON)
	if err != nil {
		return err
	}
	cfg.McPID = pid
	return s.saveConfig(ctx, serverID, cfg)
}

func (s *Service) saveConfig(ctx context.Context, serverID string, cfg ServerConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("config_json", string(raw)).Error
}

func (s *Service) AgentConnected(ctx context.Context, serverID, hostname, version string) error {
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("last_seen_at", now).Error; err != nil {
		return err
	}
	agentUpdates := map[string]any{"connected_at": now}
	if hostname != "" {
		agentUpdates["hostname"] = hostname
	}
	if version != "" {
		agentUpdates["agent_version"] = version
	}
	return s.db.WithContext(ctx).Model(&models.Agent{}).Where("server_id = ?", serverID).Updates(agentUpdates).Error
}

func (s *Service) applyServerStatusEvent(ctx context.Context, serverID string, payload []byte) {
	var status protocol.ServerStatusPayload
	if err := json.Unmarshal(payload, &status); err != nil {
		return
	}
	switch status.Status {
	case protocol.ServerStatusRunning:
		if status.PID > 0 {
			pid := status.PID
			_ = s.setMcPID(ctx, serverID, &pid)
			_ = s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("status", models.ServerStatusOnline).Error
		}
		if id := strings.TrimSpace(status.GameServerID); id != "" {
			s.setGameServerStatus(ctx, id, models.GameServerStatusRunning, "")
		}
	case protocol.ServerStatusStopped:
		if id := strings.TrimSpace(status.GameServerID); id != "" {
			s.setGameServerStatus(ctx, id, models.GameServerStatusStopped, "")
		}
		_ = s.setMcPID(ctx, serverID, nil)
		_ = s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("status", models.ServerStatusOffline).Error
	case protocol.ServerStatusCrashed:
		message := strings.TrimSpace(status.Message)
		if id := strings.TrimSpace(status.GameServerID); id != "" {
			s.setGameServerStatus(ctx, id, models.GameServerStatusError, message)
		}
		_ = s.setMcPID(ctx, serverID, nil)
		_ = s.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", serverID).Update("status", models.ServerStatusOffline).Error
		if s.hub != nil && message != "" {
			s.hub.BroadcastConsole(serverID, protocol.ConsoleOutputPayload{
				Stream:       "stderr",
				Line:         message,
				GameServerID: status.GameServerID,
			})
		}
	}
}

func (s *Service) AgentDisconnected(ctx context.Context, serverID string) error {
	return s.markMinecraftStopped(ctx, serverID)
}

func (s *Service) Start(ctx context.Context, ownerID, serverID string) error {
	server, err := s.getOwned(ctx, ownerID, serverID)
	if err != nil {
		return err
	}
	if server.AgentTokenHash == nil {
		return ErrNotDeployed
	}
	if s.hub == nil || !s.hub.IsOnline(serverID) {
		return ErrAgentOffline
	}
	cfg, err := parseConfig(server.ConfigJSON)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(protocol.ServerStartPayload{
		ServerType: server.ServerType,
		JarPath:    cfg.JarPath,
		WorkDir:    cfg.WorkDir,
		Command:    cfg.Command,
		Args:       cfg.Args,
		JVMArgs:    cfg.JVMArgs,
		ExtraArgs:  cfg.ExtraArgs,
		JavaBin:    cfg.JavaBin,
	})
	_ = s.db.WithContext(ctx).Model(server).Update("status", models.ServerStatusStarting).Error
	return s.hub.SendCommand(ctx, serverID, protocol.Envelope{
		Type:    protocol.TypeCmdServerStart,
		Payload: payload,
	})
}

func (s *Service) Stop(ctx context.Context, ownerID, serverID string) error {
	server, err := s.getOwned(ctx, ownerID, serverID)
	if err != nil {
		return err
	}
	if s.hub == nil || !s.hub.IsOnline(serverID) {
		return ErrAgentOffline
	}
	payload, _ := json.Marshal(protocol.ServerStopPayload{Graceful: true, TimeoutSec: 30})
	_ = s.db.WithContext(ctx).Model(server).Update("status", models.ServerStatusStopping).Error
	return s.hub.SendCommand(ctx, serverID, protocol.Envelope{
		Type:    protocol.TypeCmdServerStop,
		Payload: payload,
	})
}

func (s *Service) Restart(ctx context.Context, ownerID, serverID string) error {
	if err := s.Stop(ctx, ownerID, serverID); err != nil && !errors.Is(err, ErrAgentOffline) {
		return err
	}
	return s.Start(ctx, ownerID, serverID)
}

func (s *Service) SendConsoleInput(ctx context.Context, ownerID, serverID, line string) error {
	if strings.TrimSpace(line) == "" {
		return ErrValidation
	}
	if _, err := s.getOwned(ctx, ownerID, serverID); err != nil {
		return err
	}
	if s.hub == nil || !s.hub.IsOnline(serverID) {
		return ErrAgentOffline
	}
	return s.hub.SendConsoleInput(ctx, serverID, line)
}

func (s *Service) AttachConsole(ctx context.Context, ownerID, serverID, gameServerID string) error {
	if _, err := s.getOwned(ctx, ownerID, serverID); err != nil {
		return err
	}
	if s.hub == nil || !s.hub.IsOnline(serverID) {
		return ErrAgentOffline
	}
	workDir, taggedGameServerID, err := s.consoleWorkDir(ctx, serverID, gameServerID)
	if err != nil {
		return err
	}
	if workDir == "" {
		return nil
	}
	payload, err := json.Marshal(protocol.ConsoleAttachPayload{
		GameServerID: taggedGameServerID,
		WorkDir:      workDir,
	})
	if err != nil {
		return err
	}
	return s.hub.SendCommand(ctx, serverID, protocol.Envelope{
		Type:    protocol.TypeCmdConsoleAttach,
		Payload: payload,
	})
}

func (s *Service) consoleWorkDir(ctx context.Context, serverID, gameServerID string) (workDir, taggedGameServerID string, err error) {
	gameServerID = strings.TrimSpace(gameServerID)
	if gameServerID != "" {
		var item models.GameServer
		if err := s.db.WithContext(ctx).Where("id = ? AND server_id = ?", gameServerID, serverID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", "", ErrNotFound
			}
			return "", "", err
		}
		return item.WorkDir, item.ID, nil
	}
	server, err := s.getByID(ctx, serverID)
	if err != nil {
		return "", "", err
	}
	cfg, err := parseConfig(server.ConfigJSON)
	if err != nil {
		return "", "", err
	}
	return cfg.WorkDir, cfg.ActiveGameServerID, nil
}

func (s *Service) getOwned(ctx context.Context, ownerID, serverID string) (*models.Server, error) {
	server, err := s.getByID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server.OwnerID != ownerID {
		return nil, ErrForbidden
	}
	return server, nil
}

func (s *Service) getByID(ctx context.Context, serverID string) (*models.Server, error) {
	var server models.Server
	if err := s.db.WithContext(ctx).Where("id = ?", serverID).First(&server).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &server, nil
}

func (s *Service) viewFromModel(ctx context.Context, server *models.Server) (*ServerView, error) {
	cfg, err := parseConfig(server.ConfigJSON)
	if err != nil {
		return nil, err
	}
	var cred models.SSHCredential
	sshView := SSHPublicView{}
	if err := s.db.WithContext(ctx).Where("server_id = ?", server.ID).First(&cred).Error; err == nil {
		sshView = SSHPublicView{Host: cred.Host, Port: cred.Port, Username: cred.Username}
	}
	online := s.hub != nil && s.hub.IsOnline(server.ID)
	minecraftRunning := cfg.McPID != nil && *cfg.McPID > 0
	status := server.Status
	if status == models.ServerStatusOnline && !minecraftRunning {
		status = models.ServerStatusOffline
	}
	var agentVersion *string
	var agent models.Agent
	if err := s.db.WithContext(ctx).Where("server_id = ?", server.ID).First(&agent).Error; err == nil {
		agentVersion = agent.AgentVersion
	}
	return &ServerView{
		ID:               server.ID,
		Name:             server.Name,
		Slug:             server.Slug,
		ServerType:       server.ServerType,
		Status:           status,
		MCVersion:        server.MCVersion,
		Config:           cfg,
		SSH:              sshView,
		AgentDeployed:    s.isAgentDeployed(ctx, server),
		AgentOnline:      online,
		AgentVersion:     agentVersion,
		MinecraftRunning: minecraftRunning,
		LastSeenAt:       server.LastSeenAt,
		CreatedAt:        server.CreatedAt,
		UpdatedAt:        server.UpdatedAt,
	}, nil
}

func (s *Service) isAgentDeployed(ctx context.Context, server *models.Server) bool {
	if server.AgentTokenHash != nil && strings.TrimSpace(*server.AgentTokenHash) != "" {
		return true
	}
	var n int64
	if err := s.db.WithContext(ctx).Model(&models.Agent{}).Where("server_id = ?", server.ID).Count(&n).Error; err != nil {
		return false
	}
	return n > 0
}

func parseConfig(raw string) (ServerConfig, error) {
	if raw == "" {
		return ServerConfig{}, nil
	}
	var cfg ServerConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "server"
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

func uniqueSlug(ctx context.Context, db *gorm.DB, ownerID, base string) string {
	slug := base
	for i := 0; i < 100; i++ {
		var count int64
		db.WithContext(ctx).Model(&models.Server{}).Where("owner_id = ? AND slug = ?", ownerID, slug).Count(&count)
		if count == 0 {
			return slug
		}
		slug = base + "-" + uuid.NewString()[:8]
	}
	return base + "-" + uuid.NewString()
}
