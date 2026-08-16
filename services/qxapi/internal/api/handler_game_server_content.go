package api

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

type syncContentBody struct {
	mods.SyncModRequest
}

type deleteContentBody struct {
	Filename  string `json:"filename" binding:"required"`
	ModTarget string `json:"mod_target"`
}

func (h *GameServersHandler) ListContentResources(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListGameServerResources(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		c.Query("kind"),
		c.Query("mod_target"),
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *GameServersHandler) ListClientMods(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerClientMods(
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

func (h *GameServersHandler) SyncResourcepack(c *gin.Context) {
	h.syncGameServerContent(c, "resourcepack", gameServerSupportsClientContent)
}

func (h *GameServersHandler) SyncShader(c *gin.Context) {
	h.syncGameServerContent(c, "shader", gameServerSupportsClientContent)
}

func (h *GameServersHandler) DeleteMod(c *gin.Context) {
	h.deleteGameServerContent(c, "mod", gameServerSupportsMods)
}

func (h *GameServersHandler) DeletePlugin(c *gin.Context) {
	h.deleteGameServerContent(c, "plugin", gameServerSupportsPlugins)
}

func (h *GameServersHandler) DeleteDatapack(c *gin.Context) {
	h.deleteGameServerContent(c, "datapack", gameServerSupportsDatapacks)
}

func (h *GameServersHandler) DeleteResourcepack(c *gin.Context) {
	h.deleteGameServerContent(c, "resourcepack", gameServerSupportsClientContent)
}

func (h *GameServersHandler) DeleteShader(c *gin.Context) {
	h.deleteGameServerContent(c, "shader", gameServerSupportsClientContent)
}

func (h *GameServersHandler) ListClientResourcepacks(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerClientResourcepacks(
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

func (h *GameServersHandler) ListClientShaders(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerClientShaders(
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

	modTarget := strings.TrimSpace(body.ModTarget)
	entries, err := listContentEntries(
		c.Request.Context(),
		h,
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		contentKind,
		modTarget,
	)
	if err != nil {
		gameServerError(c, err)
		return
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
		modTarget,
		body.Filename,
		body.DownloadURL,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	_ = h.Service.RecordGameServerSync(c.Request.Context(), gs.ID, contentKind, body.SyncModRequest)

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
			"mod_target":     modTarget,
		},
	})
}

func (h *GameServersHandler) deleteGameServerContent(
	c *gin.Context,
	contentKind string,
	supports func(string) bool,
) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var body deleteContentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
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

	modTarget := strings.TrimSpace(body.ModTarget)
	_, err = h.Service.DeleteGameServerContent(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		contentKind,
		modTarget,
		body.Filename,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	_ = h.Service.RemoveGameServerResource(c.Request.Context(), gs.ID, body.Filename, contentKind, modTarget)
	c.JSON(http.StatusOK, gin.H{"status": "deleted", "filename": body.Filename})
}

func listContentEntries(
	ctx context.Context,
	h *GameServersHandler,
	userID, vpsID, gameServerID, contentKind, modTarget string,
) ([]protocol.FileEntry, error) {
	switch contentKind {
	case "mod":
		if strings.EqualFold(modTarget, "client-mods") {
			return h.Service.ListGameServerClientMods(ctx, userID, vpsID, gameServerID)
		}
		return h.Service.ListGameServerMods(ctx, userID, vpsID, gameServerID)
	case "plugin":
		return h.Service.ListGameServerPlugins(ctx, userID, vpsID, gameServerID)
	case "datapack":
		return h.Service.ListGameServerDatapacks(ctx, userID, vpsID, gameServerID)
	case "resourcepack":
		if strings.EqualFold(modTarget, "client-resourcepacks") {
			return h.Service.ListGameServerClientResourcepacks(ctx, userID, vpsID, gameServerID)
		}
		return h.Service.ListGameServerResourcepacks(ctx, userID, vpsID, gameServerID)
	case "shader":
		if strings.EqualFold(modTarget, "client-shaders") {
			return h.Service.ListGameServerClientShaders(ctx, userID, vpsID, gameServerID)
		}
		return h.Service.ListGameServerShaders(ctx, userID, vpsID, gameServerID)
	default:
		return nil, nil
	}
}

func (h *GameServersHandler) UploadMod(c *gin.Context) {
	h.uploadGameServerContent(c, "mod", gameServerSupportsMods, c.PostForm("mod_target"))
}

func (h *GameServersHandler) uploadGameServerContent(
	c *gin.Context,
	contentKind string,
	supports func(string) bool,
	modTarget string,
) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
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
	file, err := c.FormFile("file")
	if err != nil {
		JSONValidation(c, "missing file")
		return
	}
	if file.Size > protocol.MaxContentFileBytes {
		JSONValidation(c, "file too large")
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
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jar" && ext != ".zip" && ext != ".mrpack" {
		JSONValidation(c, "invalid file extension")
		return
	}
	result, err := h.Service.UploadGameServerContent(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		contentKind,
		strings.TrimSpace(modTarget),
		file.Filename,
		data,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	_ = h.Service.RecordGameServerUpload(c.Request.Context(), gs.ID, contentKind, result.Filename, modTarget, int64(len(data)))
	c.JSON(http.StatusCreated, gin.H{
		"status":   result.Status,
		"filename": result.Filename,
		"path":     result.RelPath,
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

func gameServerSupportsClientContent(serverType string) bool {
	switch strings.ToLower(serverType) {
	case "forge", "neoforge", "fabric", "quilt":
		return true
	default:
		return false
	}
}
