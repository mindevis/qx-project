package servers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const agentRPCTimeout = 30 * time.Second

type rpcResult struct {
	payload []byte
	err     error
}

func (s *Service) GetGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*GameServerView, error) {
	item, err := s.getOwnedGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	view := gameServerViewFromModel(item)
	return &view, nil
}

func (s *Service) GetGameServerProperties(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.PropertyEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.GameServerWorkDirPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerPropertiesGet, protocol.TypeResServerPropertiesGet, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerPropertiesResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Properties, nil
}

func (s *Service) PatchGameServerProperties(
	ctx context.Context,
	ownerID, vpsID, gameServerID string,
	updates map[string]string,
) error {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return ErrValidation
	}
	clean := make(map[string]string, len(updates))
	for key, value := range updates {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		clean[key] = value
	}
	if len(clean) == 0 {
		return ErrValidation
	}
	payload, err := json.Marshal(protocol.ServerPropertiesPatchPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Updates:      clean,
	})
	if err != nil {
		return err
	}
	_, err = s.agentRPC(ctx, vpsID, protocol.TypeCmdServerPropertiesPatch, protocol.TypeResServerPropertiesPatch, payload)
	return err
}

func (s *Service) ListGameServerFiles(ctx context.Context, ownerID, vpsID, gameServerID, path string) ([]protocol.FileEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerFilesPathPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Path:         path,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerFilesList, protocol.TypeResServerFilesList, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerFilesListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) ReadGameServerFile(ctx context.Context, ownerID, vpsID, gameServerID, path string) (*protocol.ServerFilesReadResult, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrValidation
	}
	payload, err := json.Marshal(protocol.ServerFilesPathPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Path:         path,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerFilesRead, protocol.TypeResServerFilesRead, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerFilesReadResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) WriteGameServerFile(ctx context.Context, ownerID, vpsID, gameServerID, path, content string) error {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return ErrValidation
	}
	payload, err := json.Marshal(protocol.ServerFilesWritePayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		Path:         path,
		Content:      content,
	})
	if err != nil {
		return err
	}
	_, err = s.agentRPC(ctx, vpsID, protocol.TypeCmdServerFilesWrite, protocol.TypeResServerFilesWrite, payload)
	return err
}

func (s *Service) ListGameServerClientMods(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerModsListPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		ServerType:   item.ServerType,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerClientModsList, protocol.TypeResServerClientModsList, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerModsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) listGameServerWorkDirContent(
	ctx context.Context,
	ownerID, vpsID, gameServerID, cmdType, resType string,
) ([]protocol.FileEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.GameServerWorkDirPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, cmdType, resType, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerModsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) ListGameServerResourcepacks(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	return s.listGameServerWorkDirContent(ctx, ownerID, vpsID, gameServerID, protocol.TypeCmdServerResourcepacksList, protocol.TypeResServerResourcepacksList)
}

func (s *Service) ListGameServerClientResourcepacks(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	return s.listGameServerWorkDirContent(ctx, ownerID, vpsID, gameServerID, protocol.TypeCmdServerClientResourcepacksList, protocol.TypeResServerClientResourcepacksList)
}

func (s *Service) ListGameServerShaders(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	return s.listGameServerWorkDirContent(ctx, ownerID, vpsID, gameServerID, protocol.TypeCmdServerShadersList, protocol.TypeResServerShadersList)
}

func (s *Service) ListGameServerClientShaders(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	return s.listGameServerWorkDirContent(ctx, ownerID, vpsID, gameServerID, protocol.TypeCmdServerClientShadersList, protocol.TypeResServerClientShadersList)
}

func (s *Service) ListGameServerMods(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerModsListPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		ServerType:   item.ServerType,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerModsList, protocol.TypeResServerModsList, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerModsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) ListGameServerPlugins(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.GameServerWorkDirPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerPluginsList, protocol.TypeResServerPluginsList, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerModsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) ListGameServerDatapacks(ctx context.Context, ownerID, vpsID, gameServerID string) ([]protocol.FileEntry, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.GameServerWorkDirPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerDatapacksList, protocol.TypeResServerDatapacksList, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerModsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) InstallGameServerContent(
	ctx context.Context,
	ownerID, vpsID, gameServerID, contentKind, modTarget, filename, downloadURL string,
) (*protocol.ServerContentInstallResult, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	filename = strings.TrimSpace(filename)
	downloadURL = strings.TrimSpace(downloadURL)
	contentKind = strings.ToLower(strings.TrimSpace(contentKind))
	modTarget = strings.TrimSpace(modTarget)
	if filename == "" || downloadURL == "" || contentKind == "" {
		return nil, ErrValidation
	}
	if err := validateGameServerContentKind(item.ServerType, contentKind); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerContentInstallPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		ServerType:   item.ServerType,
		ContentKind:  contentKind,
		ModTarget:    modTarget,
		Filename:     filename,
		DownloadURL:  downloadURL,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerContentInstall, protocol.TypeResServerContentInstall, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerContentInstallResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) UploadGameServerContent(
	ctx context.Context,
	ownerID, vpsID, gameServerID, contentKind, modTarget, filename string,
	data []byte,
) (*protocol.ServerContentUploadResult, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	filename = strings.TrimSpace(filename)
	contentKind = strings.ToLower(strings.TrimSpace(contentKind))
	modTarget = strings.TrimSpace(modTarget)
	if filename == "" || contentKind == "" || len(data) == 0 {
		return nil, ErrValidation
	}
	if int64(len(data)) > 32*1024*1024 {
		return nil, ErrValidation
	}
	if err := validateGameServerContentKind(item.ServerType, contentKind); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerContentUploadPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		ServerType:   item.ServerType,
		ContentKind:  contentKind,
		ModTarget:    modTarget,
		Filename:     filename,
		ContentB64:   base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerContentUpload, protocol.TypeResServerContentUpload, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerContentUploadResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) ReadGameServerContent(
	ctx context.Context,
	ownerID, vpsID, gameServerID, contentKind, modTarget, filename string,
) ([]byte, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	filename = strings.TrimSpace(filename)
	contentKind = strings.ToLower(strings.TrimSpace(contentKind))
	modTarget = strings.TrimSpace(modTarget)
	if filename == "" || contentKind == "" {
		return nil, ErrValidation
	}
	if err := validateGameServerContentKind(item.ServerType, contentKind); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerContentReadPayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		ServerType:   item.ServerType,
		ContentKind:  contentKind,
		ModTarget:    modTarget,
		Filename:     filename,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerContentRead, protocol.TypeResServerContentRead, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerContentReadResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.ContentB64 == "" {
		return nil, ErrValidation
	}
	data, err := base64.StdEncoding.DecodeString(result.ContentB64)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Service) DeleteGameServerContent(
	ctx context.Context,
	ownerID, vpsID, gameServerID, contentKind, modTarget, filename string,
) (*protocol.ServerContentDeleteResult, error) {
	item, err := s.requireInstalledGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	filename = strings.TrimSpace(filename)
	contentKind = strings.ToLower(strings.TrimSpace(contentKind))
	modTarget = strings.TrimSpace(modTarget)
	if filename == "" || contentKind == "" {
		return nil, ErrValidation
	}
	if err := validateGameServerContentKind(item.ServerType, contentKind); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(protocol.ServerContentDeletePayload{
		GameServerID: item.ID,
		WorkDir:      item.WorkDir,
		ServerType:   item.ServerType,
		ContentKind:  contentKind,
		ModTarget:    modTarget,
		Filename:     filename,
	})
	if err != nil {
		return nil, err
	}
	raw, err := s.agentRPC(ctx, vpsID, protocol.TypeCmdServerContentDelete, protocol.TypeResServerContentDelete, payload)
	if err != nil {
		return nil, err
	}
	var result protocol.ServerContentDeleteResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func validateGameServerContentKind(serverType, contentKind string) error {
	switch contentKind {
	case "mod":
		switch strings.ToLower(strings.TrimSpace(serverType)) {
		case "forge", "neoforge", "fabric", "quilt", "mohist", "magma", "arclight":
			return nil
		default:
			return ErrValidation
		}
	case "plugin":
		switch strings.ToLower(strings.TrimSpace(serverType)) {
		case "paper", "spigot", "purpur", "mohist", "magma", "arclight":
			return nil
		default:
			return ErrValidation
		}
	case "datapack":
		return nil
	case "resourcepack", "shader":
		switch strings.ToLower(strings.TrimSpace(serverType)) {
		case "forge", "neoforge", "fabric", "quilt":
			return nil
		default:
			return ErrValidation
		}
	default:
		return ErrValidation
	}
}

func (s *Service) requireInstalledGameServer(ctx context.Context, ownerID, vpsID, gameServerID string) (*models.GameServer, error) {
	item, err := s.getOwnedGameServer(ctx, ownerID, vpsID, gameServerID)
	if err != nil {
		return nil, err
	}
	if !gameServerIsInstalled(item) {
		return nil, ErrGameServerNotInstalled
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return nil, ErrAgentOffline
	}
	return item, nil
}

func (s *Service) agentRPC(ctx context.Context, vpsID, cmdType, resType string, payload []byte) ([]byte, error) {
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return nil, ErrAgentOffline
	}
	requestID := uuid.NewString()
	ch := make(chan rpcResult, 1)
	s.pendingRPC.Store(requestID, ch)
	defer s.pendingRPC.Delete(requestID)

	env := protocol.Envelope{
		V:         protocol.Version,
		Type:      cmdType,
		RequestID: requestID,
		TS:        time.Now().UTC().Format(time.RFC3339),
		Payload:   payload,
	}
	if err := s.hub.SendCommand(ctx, vpsID, env); err != nil {
		return nil, err
	}

	timer := time.NewTimer(agentRPCTimeout)
	defer timer.Stop()
	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		_ = resType
		return res.payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, agenthub.ErrTimeout
	}
}

func (s *Service) deliverRPCResponse(requestID string, payload []byte) {
	if requestID == "" {
		return
	}
	raw, ok := s.pendingRPC.Load(requestID)
	if !ok {
		return
	}
	ch := raw.(chan rpcResult)
	var errPayload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(payload, &errPayload) == nil && strings.TrimSpace(errPayload.Error) != "" {
		ch <- rpcResult{err: errors.New(errPayload.Error)}
		return
	}
	ch <- rpcResult{payload: payload}
}

func isRPCResponseType(t string) bool {
	switch t {
	case protocol.TypeResServerPropertiesGet,
		protocol.TypeResServerPropertiesPatch,
		protocol.TypeResServerFilesList,
		protocol.TypeResServerFilesRead,
		protocol.TypeResServerFilesWrite,
		protocol.TypeResServerModsList,
		protocol.TypeResServerClientModsList,
		protocol.TypeResServerResourcepacksList,
		protocol.TypeResServerClientResourcepacksList,
		protocol.TypeResServerShadersList,
		protocol.TypeResServerClientShadersList,
		protocol.TypeResServerPluginsList,
		protocol.TypeResServerDatapacksList,
		protocol.TypeResServerContentInstall,
		protocol.TypeResServerContentUpload,
		protocol.TypeResServerContentRead,
		protocol.TypeResServerContentDelete:
		return true
	default:
		return false
	}
}
