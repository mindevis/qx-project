package servers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const defaultOllamaListenAddr = "127.0.0.1:11434"
const ollamaStartTimeout = 3 * time.Minute

type OllamaView struct {
	Status       string                 `json:"status"`
	Version      string                 `json:"version,omitempty"`
	ListenAddr   string                 `json:"listen_addr,omitempty"`
	PullingModel string                 `json:"pulling_model,omitempty"`
	LastError    string                 `json:"last_error,omitempty"`
	Models       []protocol.OllamaModel `json:"models"`
}

func (s *Service) GetOllama(ctx context.Context, ownerID, vpsID string) (*OllamaView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	view := ollamaViewFromRow(row)
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return view, nil
	}
	if raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdOllamaStatus, protocol.TypeResOllamaStatus, ollamaRootPayload(row)); err == nil {
		s.mergeOllamaStatus(ctx, vpsID, raw)
		if row, err = s.getOllamaRow(ctx, vpsID); err == nil {
			view = ollamaViewFromRow(row)
		}
	}
	if view.Status == models.OllamaStatusRunning {
		if raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdOllamaModelList, protocol.TypeResOllamaModelList, ollamaRootPayload(row)); err == nil {
			var listed protocol.OllamaModelListResult
			if json.Unmarshal(raw, &listed) == nil {
				view.Models = listed.Models
				if view.Models == nil {
					view.Models = []protocol.OllamaModel{}
				}
			}
		}
	}
	return view, nil
}

func (s *Service) InstallOllama(ctx context.Context, ownerID, vpsID string) (*OllamaView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOrCreateOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if ollamaBusy(row.Status) {
		return nil, ErrOllamaBusy
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(row).Updates(map[string]any{
		"status":        models.OllamaStatusInstalling,
		"pulling_model": "",
		"last_error":    "",
		"updated_at":    now,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		V:         protocol.Version,
		Type:      protocol.TypeCmdOllamaInstall,
		RequestID: newOllamaRequestID(),
		TS:        now.Format(time.RFC3339),
		Payload:   ollamaRootPayload(row),
	}); err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return nil, err
	}
	return s.GetOllama(ctx, ownerID, vpsID)
}

func (s *Service) StartOllama(ctx context.Context, ownerID, vpsID string) (*OllamaView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status == models.OllamaStatusNotInstalled {
		return nil, ErrOllamaNotInstalled
	}
	if row.Status == models.OllamaStatusRunning {
		return nil, ErrOllamaAlreadyRunning
	}
	if ollamaBusy(row.Status) {
		return nil, ErrOllamaBusy
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.OllamaStatusStarting,
		"last_error": "",
		"updated_at": now,
	}).Error
	_, err = s.agentRPCWait(ctx, vpsID, protocol.TypeCmdOllamaStart, protocol.TypeResOllamaStart, ollamaRootPayload(row), ollamaStartTimeout)
	if err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return nil, err
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.OllamaStatusRunning,
		"updated_at": time.Now().UTC(),
	}).Error
	return s.GetOllama(ctx, ownerID, vpsID)
}

func (s *Service) StopOllama(ctx context.Context, ownerID, vpsID string) (*OllamaView, error) {
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status == models.OllamaStatusNotInstalled {
		return nil, ErrOllamaNotInstalled
	}
	now := time.Now().UTC()
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.OllamaStatusStopping,
		"updated_at": now,
	}).Error
	_, err = s.agentRPC(ctx, vpsID, protocol.TypeCmdOllamaStop, protocol.TypeResOllamaStop, ollamaRootPayload(row))
	if err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return nil, err
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":        models.OllamaStatusInstalled,
		"pulling_model": "",
		"updated_at":    time.Now().UTC(),
	}).Error
	return s.GetOllama(ctx, ownerID, vpsID)
}

func (s *Service) PullOllamaModel(ctx context.Context, ownerID, vpsID, name string) (*OllamaView, error) {
	name = strings.TrimSpace(name)
	if !validOllamaModelName(name) {
		return nil, ErrOllamaInvalidModel
	}
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status != models.OllamaStatusRunning && row.Status != models.OllamaStatusPulling {
		return nil, ErrOllamaNotRunning
	}
	if ollamaBusy(row.Status) && row.Status != models.OllamaStatusPulling {
		return nil, ErrOllamaBusy
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":        models.OllamaStatusPulling,
		"pulling_model": name,
		"last_error":    "",
		"updated_at":    now,
	}).Error; err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(protocol.OllamaModelNamePayload{Name: name})
	if err := s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		V:         protocol.Version,
		Type:      protocol.TypeCmdOllamaModelPull,
		RequestID: newOllamaRequestID(),
		TS:        now.Format(time.RFC3339),
		Payload:   payload,
	}); err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return nil, err
	}
	return s.GetOllama(ctx, ownerID, vpsID)
}

func (s *Service) DeleteOllamaModel(ctx context.Context, ownerID, vpsID, name string) (*OllamaView, error) {
	name = strings.TrimSpace(name)
	if !validOllamaModelName(name) {
		return nil, ErrOllamaInvalidModel
	}
	if _, err := s.getOwned(ctx, ownerID, vpsID); err != nil {
		return nil, err
	}
	if err := s.requireAgentOnline(ctx, vpsID); err != nil {
		return nil, err
	}
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.Status != models.OllamaStatusRunning {
		return nil, ErrOllamaNotRunning
	}
	payload, _ := json.Marshal(protocol.OllamaModelNamePayload{Name: name})
	if _, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdOllamaModelDelete, protocol.TypeResOllamaModelDelete, payload); err != nil {
		return nil, err
	}
	return s.GetOllama(ctx, ownerID, vpsID)
}

func (s *Service) applyOllamaInstallResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return
	}
	var result protocol.OllamaInstallResult
	if err := json.Unmarshal(payload, &result); err != nil {
		_ = s.setOllamaError(ctx, vpsID, "invalid ollama install result")
		return
	}
	now := time.Now().UTC()
	listen := strings.TrimSpace(result.ListenAddr)
	if listen == "" {
		listen = defaultOllamaListenAddr
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":      models.OllamaStatusStarting,
		"version":     result.Version,
		"bin_path":    result.BinPath,
		"root_dir":    result.RootDir,
		"models_dir":  result.ModelsDir,
		"listen_addr": listen,
		"last_error":  "",
		"updated_at":  now,
	}).Error
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return
	}
	_ = s.hub.SendCommand(ctx, vpsID, protocol.Envelope{
		V:         protocol.Version,
		Type:      protocol.TypeCmdOllamaStart,
		RequestID: newOllamaRequestID(),
		TS:        now.Format(time.RFC3339),
		Payload:   ollamaRootPayload(row),
	})
}

func (s *Service) applyOllamaStartResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return
	}
	s.mergeOllamaStatus(ctx, vpsID, payload)
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":     models.OllamaStatusRunning,
		"last_error": "",
		"updated_at": time.Now().UTC(),
	}).Error
}

func (s *Service) applyOllamaStopResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":        models.OllamaStatusInstalled,
		"pulling_model": "",
		"last_error":    "",
		"updated_at":    time.Now().UTC(),
	}).Error
}

func (s *Service) applyOllamaPullResult(ctx context.Context, vpsID string, payload []byte) {
	if err := agentPayloadError(payload); err != nil {
		_ = s.setOllamaError(ctx, vpsID, err.Error())
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":        models.OllamaStatusRunning,
		"pulling_model": "",
		"last_error":    "",
		"updated_at":    time.Now().UTC(),
	}).Error
}

func (s *Service) mergeOllamaStatus(ctx context.Context, vpsID string, raw []byte) {
	var st protocol.OllamaStatusResult
	if json.Unmarshal(raw, &st) != nil {
		return
	}
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return
	}
	if ollamaBusy(row.Status) {
		return
	}
	status := models.OllamaStatusNotInstalled
	if st.Running {
		status = models.OllamaStatusRunning
	} else if st.Installed {
		status = models.OllamaStatusInstalled
	}
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if strings.TrimSpace(st.Version) != "" {
		updates["version"] = st.Version
	}
	if strings.TrimSpace(st.BinPath) != "" {
		updates["bin_path"] = st.BinPath
	}
	if strings.TrimSpace(st.RootDir) != "" {
		updates["root_dir"] = st.RootDir
	}
	if strings.TrimSpace(st.ModelsDir) != "" {
		updates["models_dir"] = st.ModelsDir
	}
	if strings.TrimSpace(st.ListenAddr) != "" {
		updates["listen_addr"] = st.ListenAddr
	}
	_ = s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(updates).Error
}

func (s *Service) getOllamaRow(ctx context.Context, vpsID string) (*models.ServerOllama, error) {
	var row models.ServerOllama
	err := s.db.WithContext(ctx).Where("server_id = ?", vpsID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &models.ServerOllama{
			ServerID:   vpsID,
			Status:     models.OllamaStatusNotInstalled,
			ListenAddr: defaultOllamaListenAddr,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) getOrCreateOllamaRow(ctx context.Context, vpsID string) (*models.ServerOllama, error) {
	row, err := s.getOllamaRow(ctx, vpsID)
	if err != nil {
		return nil, err
	}
	if row.CreatedAt.IsZero() && row.Status == models.OllamaStatusNotInstalled {
		now := time.Now().UTC()
		row.CreatedAt = now
		row.UpdatedAt = now
		if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
			existing, loadErr := s.getOllamaRow(ctx, vpsID)
			if loadErr == nil && !existing.CreatedAt.IsZero() {
				return existing, nil
			}
			return nil, err
		}
	}
	return row, nil
}

func (s *Service) setOllamaError(ctx context.Context, vpsID, message string) error {
	return s.db.WithContext(ctx).Model(&models.ServerOllama{}).Where("server_id = ?", vpsID).Updates(map[string]any{
		"status":        models.OllamaStatusError,
		"pulling_model": "",
		"last_error":    message,
		"updated_at":    time.Now().UTC(),
	}).Error
}

func ollamaViewFromRow(row *models.ServerOllama) *OllamaView {
	if row == nil {
		return &OllamaView{Status: models.OllamaStatusNotInstalled, Models: []protocol.OllamaModel{}}
	}
	return &OllamaView{
		Status:       row.Status,
		Version:      row.Version,
		ListenAddr:   row.ListenAddr,
		PullingModel: row.PullingModel,
		LastError:    row.LastError,
		Models:       []protocol.OllamaModel{},
	}
}

func ollamaRootPayload(row *models.ServerOllama) []byte {
	payload := protocol.OllamaControlPayload{}
	if row != nil {
		payload.RootDir = row.RootDir
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func ollamaBusy(status string) bool {
	switch status {
	case models.OllamaStatusInstalling, models.OllamaStatusStarting, models.OllamaStatusStopping, models.OllamaStatusPulling:
		return true
	default:
		return false
	}
}

func validOllamaModelName(name string) bool {
	if name == "" || len(name) > 200 || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '.', '_', '-', ':', '/':
			continue
		default:
			return false
		}
	}
	return true
}

func newOllamaRequestID() string {
	return uuid.NewString()
}
