package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

type syncContentBody struct {
	mods.SyncModRequest
}

func (h *GameServersHandler) ListPlugins(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerPlugins(
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

func (h *GameServersHandler) ListDatapacks(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerDatapacks(
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

func (h *GameServersHandler) SyncMod(c *gin.Context) {
	h.syncGameServerContent(c, "mod", gameServerSupportsMods)
}

func (h *GameServersHandler) SyncPlugin(c *gin.Context) {
	h.syncGameServerContent(c, "plugin", gameServerSupportsPlugins)
}

func (h *GameServersHandler) SyncDatapack(c *gin.Context) {
	h.syncGameServerContent(c, "datapack", gameServerSupportsDatapacks)
}

func (h *GameServersHandler) syncGameServerContent(
	c *gin.Context,
	contentKind string,
	supports func(string) bool,
) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var body syncContentBody
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
	if !supports(gs.ServerType) {
		JSONError(c, http.StatusForbidden, "CONTENT_NOT_ALLOWED", "this server type does not support "+contentKind)
		return
	}

	var entries []struct {
		Name string `json:"name"`
		Dir  bool   `json:"dir"`
	}
	switch contentKind {
	case "mod":
		modEntries, listErr := h.Service.ListGameServerMods(
			c.Request.Context(),
			userID.(string),
			c.Param("id"),
			c.Param("gameServerId"),
		)
		if listErr != nil {
			gameServerError(c, listErr)
			return
		}
		for _, entry := range modEntries {
			entries = append(entries, struct {
				Name string `json:"name"`
				Dir  bool   `json:"dir"`
			}{Name: entry.Name, Dir: entry.Dir})
		}
	case "plugin":
		pluginEntries, listErr := h.Service.ListGameServerPlugins(
			c.Request.Context(),
			userID.(string),
			c.Param("id"),
			c.Param("gameServerId"),
		)
		if listErr != nil {
			gameServerError(c, listErr)
			return
		}
		for _, entry := range pluginEntries {
			entries = append(entries, struct {
				Name string `json:"name"`
				Dir  bool   `json:"dir"`
			}{Name: entry.Name, Dir: entry.Dir})
		}
	case "datapack":
		datapackEntries, listErr := h.Service.ListGameServerDatapacks(
			c.Request.Context(),
			userID.(string),
			c.Param("id"),
			c.Param("gameServerId"),
		)
		if listErr != nil {
			gameServerError(c, listErr)
			return
		}
		for _, entry := range datapackEntries {
			entries = append(entries, struct {
				Name string `json:"name"`
				Dir  bool   `json:"dir"`
			}{Name: entry.Name, Dir: entry.Dir})
		}
	}
	for _, entry := range entries {
		if entry.Dir {
			continue
		}
		if strings.EqualFold(entry.Name, body.Filename) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "already_installed",
				"message": contentKind + " file already exists on server",
			})
			return
		}
	}

	result, err := h.Service.InstallGameServerContent(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		contentKind,
		body.Filename,
		body.DownloadURL,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   "installed",
		"message":  contentKind + " installed on server",
		"filename": result.Filename,
		"path":     result.RelPath,
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

func gameServerSupportsPlugins(serverType string) bool {
	switch strings.ToLower(serverType) {
	case "paper", "spigot", "purpur", "mohist", "magma", "arclight":
		return true
	default:
		return false
	}
}

func gameServerSupportsDatapacks(_ string) bool {
	return true
}
