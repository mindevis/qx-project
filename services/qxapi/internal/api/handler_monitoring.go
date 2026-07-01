package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

type MonitoringHandler struct {
	Service *servers.Service
}

type rateMonitoringRequest struct {
	Rating int `json:"rating" binding:"required"`
}

type setInstanceBindingRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

func (h *MonitoringHandler) ListBindings(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListInstanceBindings(c.Request.Context(), userID.(string))
	if err != nil {
		monitoringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MonitoringHandler) SetBinding(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req setInstanceBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.SetInstanceBinding(c.Request.Context(), userID.(string), c.Param("id"), req.InstanceID)
	if err != nil {
		monitoringError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MonitoringHandler) ClearBinding(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if err := h.Service.ClearInstanceBinding(c.Request.Context(), userID.(string), c.Param("id")); err != nil {
		monitoringError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *MonitoringHandler) ListBindable(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListBindableServers(c.Request.Context(), userID.(string), servers.ListMonitoringInput{
		MCVersion: c.Query("mc_version"),
		Loader:    c.Query("loader"),
	})
	if err != nil {
		monitoringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MonitoringHandler) List(c *gin.Context) {
	items, err := h.Service.ListMonitoringServers(c.Request.Context(), servers.ListMonitoringInput{
		MCVersion: c.Query("mc_version"),
		Loader:    c.Query("loader"),
		Mod:       c.Query("mod"),
		Plugin:    c.Query("plugin"),
		Query:     c.Query("q"),
	})
	if err != nil {
		monitoringError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MonitoringHandler) Like(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.LikeMonitoringServer(c.Request.Context(), userID.(string), c.Param("id"))
	if err != nil {
		monitoringError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *MonitoringHandler) Rate(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req rateMonitoringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.RateMonitoringServer(c.Request.Context(), userID.(string), c.Param("id"), req.Rating)
	if err != nil {
		monitoringError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func monitoringError(c *gin.Context, err error) {
	switch err {
	case servers.ErrValidation:
		JSONValidation(c, "invalid request")
	case servers.ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		slog.Error("monitoring request failed", "error", err)
		JSONInternal(c)
	}
}
