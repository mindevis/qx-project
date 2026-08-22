package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type GameServersHandler struct {
	Service *servers.Service
}

type createGameServerRequest struct {
	Name                  string   `json:"name" binding:"required"`
	ServerType            string   `json:"server_type" binding:"required"`
	MCVersion             string   `json:"mc_version" binding:"required"`
	LoaderVersion         string   `json:"loader_version"`
	Address               string   `json:"address"`
	Port                  int      `json:"port"`
	ShowInMonitoring      bool     `json:"show_in_monitoring"`
	MonitoringDescription string   `json:"monitoring_description"`
	BannerURL             string   `json:"banner_url"`
	MonitoringTags        []string `json:"monitoring_tags"`
}

type updateGameServerRequest struct {
	Name                  *string   `json:"name"`
	Address               *string   `json:"address"`
	Port                  *int      `json:"port"`
	ShowInMonitoring      *bool     `json:"show_in_monitoring"`
	MonitoringDescription *string   `json:"monitoring_description"`
	BannerURL             *string   `json:"banner_url"`
	MonitoringTags        []string  `json:"monitoring_tags"`
	MinMemoryMB           *int      `json:"min_memory_mb"`
	MaxMemoryMB           *int      `json:"max_memory_mb"`
	ExtraJVMArgs          *[]string `json:"extra_jvm_args"`
	ExtraArgs             *[]string `json:"extra_args"`
}

type changeGameServerVersionRequest struct {
	MCVersion     string `json:"mc_version" binding:"required"`
	LoaderVersion string `json:"loader_version"`
}

func (h *GameServersHandler) List(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListGameServers(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *GameServersHandler) Create(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req createGameServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.CreateGameServer(c.Request.Context(), userID.(string), c.Param("id"), servers.CreateGameServerInput{
		Name:                  req.Name,
		ServerType:            req.ServerType,
		MCVersion:             req.MCVersion,
		LoaderVersion:         req.LoaderVersion,
		Address:               req.Address,
		Port:                  req.Port,
		ShowInMonitoring:      req.ShowInMonitoring,
		MonitoringDescription: req.MonitoringDescription,
		BannerURL:             req.BannerURL,
		MonitoringTags:        req.MonitoringTags,
	})
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *GameServersHandler) Clone(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.CloneGameServer(c.Request.Context(), userID.(string), c.Param("id"), c.Param("gameServerId"))
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *GameServersHandler) Update(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req updateGameServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateGameServer(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		servers.UpdateGameServerInput{
			Name:                  req.Name,
			Address:               req.Address,
			Port:                  req.Port,
			ShowInMonitoring:      req.ShowInMonitoring,
			MonitoringDescription: req.MonitoringDescription,
			BannerURL:             req.BannerURL,
			MonitoringTags:        req.MonitoringTags,
			MinMemoryMB:           req.MinMemoryMB,
			MaxMemoryMB:           req.MaxMemoryMB,
			ExtraJVMArgs:          req.ExtraJVMArgs,
			ExtraArgs:             req.ExtraArgs,
		},
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) Delete(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	err := h.Service.DeleteGameServer(c.Request.Context(), userID.(string), c.Param("id"), c.Param("gameServerId"))
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *GameServersHandler) ChangeVersion(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req changeGameServerVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.ChangeGameServerVersion(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		servers.ChangeGameServerVersionInput{
			MCVersion:     req.MCVersion,
			LoaderVersion: req.LoaderVersion,
		},
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) Reinstall(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.ReinstallGameServer(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) Start(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.StartGameServer(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) Stop(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.StopGameServer(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) Restart(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.RestartGameServer(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) Get(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.GetGameServer(c.Request.Context(), userID.(string), c.Param("id"), c.Param("gameServerId"))
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *GameServersHandler) GetProperties(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	props, err := h.Service.GetGameServerProperties(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"properties": props})
}

func (h *GameServersHandler) PatchProperties(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req struct {
		Updates map[string]string `json:"updates" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	err := h.Service.PatchGameServerProperties(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		req.Updates,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *GameServersHandler) ListMods(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerMods(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

func (h *GameServersHandler) ListFiles(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerFiles(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		c.Query("path"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": entries})
}

func (h *GameServersHandler) ReadFile(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	path := c.Query("path")
	if path == "" {
		JSONValidation(c, "path required")
		return
	}
	result, err := h.Service.ReadGameServerFile(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		path,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *GameServersHandler) WriteFile(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	path := c.Query("path")
	if path == "" {
		JSONValidation(c, "path required")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	err := h.Service.WriteGameServerFile(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		path,
		req.Content,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type mkdirFileBody struct {
	Path string `json:"path" binding:"required"`
}

func (h *GameServersHandler) MkdirFile(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var body mkdirFileBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	path, err := validateFileManagerPath(body.Path)
	if err != nil {
		JSONValidation(c, err.Error())
		return
	}
	err = h.Service.MkdirGameServerFile(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		path,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok", "path": path})
}

func (h *GameServersHandler) UploadFile(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	dir := strings.TrimSpace(c.Query("path"))
	if dir != "" {
		cleaned, err := validateFileManagerPath(dir)
		if err != nil {
			JSONValidation(c, err.Error())
			return
		}
		dir = cleaned
	}
	file, err := c.FormFile("file")
	if err != nil {
		JSONValidation(c, "missing file")
		return
	}
	if file.Size > protocol.MaxContentFileBytes {
		JSONValidation(c, "file too large")
		return
	}
	name, err := sanitizeFileManagerName(file.Filename)
	if err != nil {
		JSONValidation(c, err.Error())
		return
	}
	f, err := file.Open()
	if err != nil {
		JSONInternal(c)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, protocol.MaxContentFileBytes+1))
	if err != nil {
		JSONInternal(c)
		return
	}
	if int64(len(data)) > protocol.MaxContentFileBytes {
		JSONValidation(c, "file too large")
		return
	}
	path := joinFileManagerPath(dir, name)
	err = h.Service.UploadGameServerFile(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		path,
		data,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok", "path": path, "filename": name})
}

func (h *GameServersHandler) DeleteFile(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	path := c.Query("path")
	if path == "" {
		JSONValidation(c, "path required")
		return
	}
	err := h.Service.DeleteGameServerFile(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		path,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func gameServerError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, servers.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "not found")
	case errors.Is(err, servers.ErrForbidden):
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, servers.ErrValidation):
		JSONValidation(c, "invalid game server data")
	case errors.Is(err, servers.ErrNotDeployed):
		JSONError(c, http.StatusConflict, "NOT_DEPLOYED", "deploy agent first")
	case errors.Is(err, servers.ErrAgentOffline):
		JSONError(c, http.StatusConflict, "AGENT_OFFLINE", "agent is not connected")
	case errors.Is(err, servers.ErrGameServerBusy):
		JSONError(c, http.StatusConflict, "GAME_SERVER_BUSY", "game server is installing or starting")
	case errors.Is(err, servers.ErrGameServerNotInstalled):
		JSONValidation(c, "game server is not installed")
	case errors.Is(err, servers.ErrGameServerNotRunning):
		JSONError(c, http.StatusConflict, "GAME_SERVER_NOT_RUNNING", "game server is not running")
	case errors.Is(err, servers.ErrGameServerAlreadyRunning):
		JSONError(c, http.StatusConflict, "GAME_SERVER_ALREADY_RUNNING", "game server is already running")
	case errors.Is(err, agenthub.ErrTimeout):
		JSONError(c, http.StatusGatewayTimeout, "AGENT_TIMEOUT", "agent did not respond in time")
	case isContentInstallError(err):
		JSONError(c, http.StatusBadGateway, "CONTENT_INSTALL_FAILED", err.Error())
	case isFileManagerClientError(err):
		JSONValidation(c, err.Error())
	default:
		JSONInternal(c)
	}
}

func isFileManagerClientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "path is a") ||
		strings.Contains(msg, "invalid path") ||
		strings.Contains(msg, "invalid filename") ||
		strings.Contains(msg, "content too large") ||
		strings.Contains(msg, "file too large") ||
		strings.Contains(msg, "cannot delete")
}

func isContentInstallError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "download") ||
		strings.Contains(msg, "invalid filename") ||
		strings.Contains(msg, "rel_path")
}

func validateFileManagerPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Trim(path, "/")
	if path == "" || path == "." {
		return "", fmt.Errorf("path required")
	}
	parts := strings.Split(path, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", fmt.Errorf("invalid path")
		}
		if strings.ContainsAny(part, `/\`) {
			return "", fmt.Errorf("invalid path")
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return "", fmt.Errorf("path required")
	}
	return strings.Join(clean, "/"), nil
}

func sanitizeFileManagerName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", fmt.Errorf("invalid filename")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid filename")
	}
	return name, nil
}

func joinFileManagerPath(dir, name string) string {
	dir = strings.Trim(strings.ReplaceAll(strings.TrimSpace(dir), "\\", "/"), "/")
	if dir == "" {
		return name
	}
	return dir + "/" + name
}
