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
		writeModsUpstreamError(c, err)
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
		writeModsUpstreamError(c, err)
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
		writeModsUpstreamError(c, err)
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
		writeModsUpstreamError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": versions})
}

func (h *ModsHandler) GetVersion(c *gin.Context) {
	if h.Service == nil {
		JSONError(c, http.StatusServiceUnavailable, "MODS_UNAVAILABLE", "mods service not configured")
		return
	}
	version, err := h.Service.GetVersion(
		c.Request.Context(),
		c.Param("source"),
		c.Param("projectId"),
		c.Param("versionId"),
		c.Query("loader"),
		c.Query("mc_version"),
	)
	if err != nil {
		writeModsUpstreamError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func writeModsUpstreamError(c *gin.Context, err error) {
	msg := err.Error()
	if strings.Contains(msg, "not configured") {
		JSONError(c, http.StatusServiceUnavailable, "SOURCE_UNAVAILABLE", msg)
		return
	}
	if strings.HasPrefix(msg, "curseforge:") {
		code := "CURSEFORGE_UNAVAILABLE"
		if strings.Contains(msg, "status 403") {
			code = "CURSEFORGE_INVALID_KEY"
		}
		JSONError(c, http.StatusBadGateway, code, msg)
		return
	}
	JSONError(c, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", msg)
}
