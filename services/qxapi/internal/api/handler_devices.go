package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

type DevicesHandler struct {
	Service *launcher.Service
}

type registerDeviceRequest struct {
	DeviceID        string `json:"device_id" binding:"required"`
	OS              string `json:"os"`
	Hostname        string `json:"hostname"`
	LauncherVersion string `json:"launcher_version"`
}

func (h *DevicesHandler) Register(c *gin.Context) {
	var req registerDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	result, err := h.Service.RegisterDevice(c.Request.Context(), launcher.RegisterDeviceInput{
		DeviceID:        req.DeviceID,
		OS:              req.OS,
		Hostname:        req.Hostname,
		LauncherVersion: req.LauncherVersion,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid device data")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *DevicesHandler) Status(c *gin.Context) {
	result, err := h.Service.DeviceStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
			return
		}
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid device id")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, result)
}

type linkDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

func (h *DevicesHandler) Link(c *gin.Context) {
	var req linkDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}

	in := launcher.LinkDeviceInput{
		DeviceID: req.DeviceID,
	}
	if userID, ok := c.Get(UserIDKey); ok {
		in.UserID = userID.(string)
	}

	result, err := h.Service.LinkDevice(c.Request.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, launcher.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
		case errors.Is(err, launcher.ErrLinkExpired):
			JSONError(c, http.StatusGone, "LINK_EXPIRED", "link request expired")
		case errors.Is(err, launcher.ErrDeviceNotPending):
			JSONError(c, http.StatusConflict, "CONFLICT", "device is not pending link")
		case errors.Is(err, launcher.ErrValidation):
			JSONValidation(c, "invalid link data")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *DevicesHandler) MeInstances(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListInstancesForDevice(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, launcher.ErrDeviceNotLinked) {
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked")
			return
		}
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
			return
		}
		JSONInternal(c)
		return
	}
	out := make([]instanceResponse, 0, len(items))
	for _, item := range items {
		out = append(out, instanceFromModel(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *DevicesHandler) Me(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	info, err := h.Service.DeviceMe(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *DevicesHandler) Unlink(c *gin.Context) {
	if deviceID, ok := deviceIDFromContext(c); ok {
		result, err := h.Service.UnlinkDevice(c.Request.Context(), deviceID)
		if err != nil {
			switch {
			case errors.Is(err, launcher.ErrNotFound):
				JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
			case errors.Is(err, launcher.ErrDeviceNotLinked):
				JSONError(c, http.StatusConflict, "CONFLICT", "device not linked")
			case errors.Is(err, launcher.ErrValidation):
				JSONValidation(c, "invalid device id")
			default:
				JSONInternal(c)
			}
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	result, err := h.Service.UnlinkDeviceForOwner(c.Request.Context(), owner)
	if err != nil {
		switch {
		case errors.Is(err, launcher.ErrNotFound):
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "linked device not found")
		case errors.Is(err, launcher.ErrDeviceNotLinked):
			JSONError(c, http.StatusConflict, "CONFLICT", "device not linked")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *DevicesHandler) UserLinkedDevice(c *gin.Context) {
	userID, ok := c.Get(UserIDKey)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	info, err := h.Service.UserLinkedDevice(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			c.JSON(http.StatusOK, gin.H{"linked": false})
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"linked":     true,
		"device_id":  info.DeviceID,
		"status":     info.Status,
		"owner_type": info.OwnerType,
	})
}
