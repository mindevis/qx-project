package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

type ModsHandler struct {
	Service *mods.Service
}

func (h *ModsHandler) Search(c *gin.Context) {
	if h.Service == nil {
		JSONError(c, http.StatusServiceUnavailable, "MODS_UNAVAILABLE", "mods service not configured")
		return
	}
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		JSONValidation(c, "q required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.Service.Search(
		c.Request.Context(),
		query,
		c.DefaultQuery("type", mods.ProjectTypeMod),
		c.Query("loader"),
		c.Query("mc_version"),
		limit,
	)
	if err != nil {
		JSONError(c, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":              items,
		"curseforge_enabled": h.Service.CurseForgeEnabled(),
	})
}

func (h *ModsHandler) Browse(c *gin.Context) {
	if h.Service == nil {
		JSONError(c, http.StatusServiceUnavailable, "MODS_UNAVAILABLE", "mods service not configured")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, hasMore, err := h.Service.Browse(
		c.Request.Context(),
		c.DefaultQuery("type", mods.ProjectTypeMod),
		c.Query("loader"),
		c.Query("mc_version"),
		c.DefaultQuery("source", "all"),
		c.DefaultQuery("sort", "downloads"),
		limit,
		offset,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not configured") {
			JSONError(c, http.StatusServiceUnavailable, "SOURCE_UNAVAILABLE", err.Error())
			return
		}
		JSONError(c, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":              items,
		"has_more":           hasMore,
		"curseforge_enabled": h.Service.CurseForgeEnabled(),
	})
}

func (h *ModsHandler) GetProject(c *gin.Context) {
	if h.Service == nil {
		JSONError(c, http.StatusServiceUnavailable, "MODS_UNAVAILABLE", "mods service not configured")
		return
	}
	project, err := h.Service.GetProject(c.Request.Context(), c.Param("source"), c.Param("projectId"))
	if err != nil {
		if strings.Contains(err.Error(), "not configured") {
			JSONError(c, http.StatusServiceUnavailable, "SOURCE_UNAVAILABLE", err.Error())
			return
		}
		JSONError(c, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", err.Error())
		return
	}
	c.JSON(http.StatusOK, project)
}

func (h *ModsHandler) ListVersions(c *gin.Context) {
	if h.Service == nil {
		JSONError(c, http.StatusServiceUnavailable, "MODS_UNAVAILABLE", "mods service not configured")
		return
	}
	versions, err := h.Service.ListVersions(
		c.Request.Context(),
		c.Param("source"),
		c.Param("projectId"),
		c.Query("loader"),
		c.Query("mc_version"),
	)
	if err != nil {
		JSONError(c, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": versions})
}

type syncModBody struct {
	mods.SyncModRequest
}

func (h *GameServersHandler) SyncMod(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var body syncModBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	if body.Source == "" || body.ProjectID == "" || body.VersionID == "" || body.Filename == "" || body.DownloadURL == "" {
		JSONValidation(c, "source, project_id, version_id, filename, and download_url are required")
		return
	}

	gs, err := h.Service.GetGameServer(c.Request.Context(), userID.(string), c.Param("id"), c.Param("gameServerId"))
	if err != nil {
		gameServerError(c, err)
		return
	}
	if !gameServerSupportsMods(gs.ServerType) {
		JSONError(c, http.StatusForbidden, "CONTENT_NOT_ALLOWED", "this server type does not support mods")
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
	for _, entry := range entries {
		if strings.EqualFold(entry.Name, body.Filename) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "already_installed",
				"message": "mod file already exists on server",
			})
			return
		}
	}

	// Agent cmd.mods.install is planned (see docs/server-content-install.md).
	c.JSON(http.StatusAccepted, gin.H{
		"status":  "queued",
		"message": "mod sync queued; agent install pipeline not yet implemented",
		"item": gin.H{
			"source":         body.Source,
			"project_id":     body.ProjectID,
			"version_id":     body.VersionID,
			"filename":       body.Filename,
			"download_url":   body.DownloadURL,
			"project_name":   body.ProjectName,
			"version_number": body.VersionNumber,
		},
	})
}

func gameServerSupportsMods(serverType string) bool {
	switch strings.ToLower(serverType) {
	case "forge", "neoforge", "fabric", "quilt", "mohist", "magma", "arclight":
		return true
	default:
		return false
	}
}
