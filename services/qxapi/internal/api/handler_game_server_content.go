package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
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

type patchContentResourceBody struct {
	Filename     string `json:"filename" binding:"required"`
	ResourceType string `json:"resource_type"`
	SideOverride string `json:"side_override" binding:"required"`
}

type installContentURLBody struct {
	URL      string `json:"url" binding:"required"`
	Filename string `json:"filename"`
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

func (h *GameServersHandler) PatchContentResource(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var body patchContentResourceBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	err := h.Service.UpdateGameServerResourceSide(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		body.Filename,
		body.ResourceType,
		body.SideOverride,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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

func (h *GameServersHandler) ListResourcepacks(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerResourcepacks(
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

func (h *GameServersHandler) ListShaders(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	entries, err := h.Service.ListGameServerShaders(
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
	installed, err := contentAlreadyInstalled(
		c.Request.Context(),
		h,
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
	if installed {
		c.JSON(http.StatusOK, gin.H{
			"status":  "already_installed",
			"message": contentKind + " file already exists on server",
		})
		return
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
		false,
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

func contentAlreadyInstalled(
	ctx context.Context,
	h *GameServersHandler,
	userID, vpsID, gameServerID, contentKind, modTarget, filename string,
) (bool, error) {
	targets := []string{modTarget}
	switch contentKind {
	case "mod":
		if strings.EqualFold(modTarget, "client-mods") {
			targets = append(targets, "")
		} else {
			targets = append(targets, "client-mods")
		}
	case "resourcepack":
		if strings.EqualFold(modTarget, "client-resourcepacks") {
			targets = append(targets, "")
		} else {
			targets = append(targets, "client-resourcepacks")
		}
	case "shader":
		if strings.EqualFold(modTarget, "client-shaders") {
			targets = append(targets, "")
		} else {
			targets = append(targets, "client-shaders")
		}
	}
	for _, target := range targets {
		entries, err := listContentEntries(ctx, h, userID, vpsID, gameServerID, contentKind, target)
		if err != nil {
			return false, err
		}
		for _, entry := range entries {
			if !entry.Dir && strings.EqualFold(entry.Name, filename) {
				return true, nil
			}
		}
	}
	return false, nil
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

func (h *GameServersHandler) UploadPlugin(c *gin.Context) {
	h.uploadGameServerContent(c, "plugin", gameServerSupportsPlugins, "")
}

func (h *GameServersHandler) InstallPluginFromURL(c *gin.Context) {
	h.installGameServerContentFromURL(c, "plugin", gameServerSupportsPlugins)
}

func (h *GameServersHandler) installGameServerContentFromURL(
	c *gin.Context,
	contentKind string,
	supports func(string) bool,
) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var body installContentURLBody
	if err := c.ShouldBindJSON(&body); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	downloadURL, filename, err := parseUserContentDownload(body.URL, body.Filename)
	if err != nil {
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

	installed, err := contentAlreadyInstalled(
		c.Request.Context(),
		h,
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		contentKind,
		"",
		filename,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	if installed {
		c.JSON(http.StatusOK, gin.H{
			"status":  "already_installed",
			"message": contentKind + " file already exists on server",
		})
		return
	}

	result, err := h.Service.InstallGameServerContent(
		c.Request.Context(),
		userID.(string),
		c.Param("id"),
		c.Param("gameServerId"),
		contentKind,
		"",
		filename,
		downloadURL,
		true,
	)
	if err != nil {
		gameServerError(c, err)
		return
	}
	_ = h.Service.RecordGameServerUpload(
		c.Request.Context(),
		gs.ID,
		contentKind,
		result.Filename,
		"",
		"",
		0,
	)
	c.JSON(http.StatusOK, gin.H{
		"status":   "installed",
		"message":  contentKind + " installed on server",
		"filename": result.Filename,
		"path":     result.RelPath,
	})
}

func parseUserContentDownload(raw, filenameOverride string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return "", "", fmt.Errorf("invalid download url")
	}
	if parsed.User != nil {
		return "", "", fmt.Errorf("invalid download url")
	}
	if strings.ToLower(parsed.Scheme) != "https" {
		return "", "", fmt.Errorf("download url must be https")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" || net.ParseIP(host) != nil {
		return "", "", fmt.Errorf("download host not allowed")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") ||
		strings.HasSuffix(host, ".lan") {
		return "", "", fmt.Errorf("download host not allowed")
	}
	filename := strings.TrimSpace(filenameOverride)
	if filename == "" {
		filename = path.Base(parsed.Path)
		if decoded, unescapeErr := url.PathUnescape(filename); unescapeErr == nil {
			filename = decoded
		}
	}
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return "", "", fmt.Errorf("filename required")
	}
	if strings.ContainsAny(filename, `/\`) {
		return "", "", fmt.Errorf("invalid filename")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jar" && ext != ".zip" {
		return "", "", fmt.Errorf("invalid file extension")
	}
	return raw, filename, nil
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
	if contentKind == "plugin" {
		if ext != ".jar" && ext != ".zip" {
			JSONValidation(c, "invalid file extension")
			return
		}
	} else if ext != ".jar" && ext != ".zip" && ext != ".mrpack" {
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
	_ = h.Service.RecordGameServerUpload(
		c.Request.Context(),
		gs.ID,
		contentKind,
		result.Filename,
		modTarget,
		c.PostForm("side_override"),
		int64(len(data)),
	)
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
	case "paper", "spigot", "purpur", "mohist", "magma", "arclight", "velocity", "waterfall", "bungeecord":
		return true
	default:
		return false
	}
}

func gameServerSupportsDatapacks(serverType string) bool {
	return !gameServerIsProxy(serverType)
}

func gameServerSupportsClientContent(serverType string) bool {
	return !gameServerIsProxy(serverType)
}

func gameServerIsProxy(serverType string) bool {
	switch strings.ToLower(strings.TrimSpace(serverType)) {
	case "velocity", "waterfall", "bungeecord":
		return true
	default:
		return false
	}
}
