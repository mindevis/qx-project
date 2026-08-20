package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

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
	Name                  *string  `json:"name"`
	Address               *string  `json:"address"`
	Port                  *int     `json:"port"`
	ShowInMonitoring      *bool    `json:"show_in_monitoring"`
	MonitoringDescription *string  `json:"monitoring_description"`
	BannerURL             *string  `json:"banner_url"`
	MonitoringTags        []string `json:"monitoring_tags"`
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
	default:
		JSONInternal(c)
	}
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
