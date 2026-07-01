package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

type UpdateRequestsHandler struct {
	Service *launcher.Service
}

type completeUpdateBody struct {
	Status          string `json:"status" binding:"required"`
	LauncherVersion string `json:"launcher_version"`
	ErrorCode       string `json:"error_code"`
}

func (h *UpdateRequestsHandler) Create(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	result, err := h.Service.RequestLauncherUpdate(c.Request.Context(), userID.(string))
	if err != nil {
		switch {
		case errors.Is(err, launcher.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "linked device not found")
		case errors.Is(err, launcher.ErrDeviceNotLinked):
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *UpdateRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingUpdate(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid device id")
			return
		}
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
			return
		}
		JSONInternal(c)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": view})
}

func (h *UpdateRequestsHandler) Complete(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if c.Param("id") != deviceID {
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "update request mismatch")
		return
	}
	var req completeUpdateBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	err := h.Service.CompleteLauncherUpdate(c.Request.Context(), deviceID, launcher.CompleteUpdateInput{
		Status:          req.Status,
		LauncherVersion: req.LauncherVersion,
		ErrorCode:       req.ErrorCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, launcher.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "update request not found")
		case errors.Is(err, launcher.ErrValidation):
			JSONValidation(c, "invalid update request")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}
