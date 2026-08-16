package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

type ModInstallRequestsHandler struct {
	Service *launcher.Service
	Tokens  *auth.TokenService
}

type createModInstallRequestBody struct {
	InstanceID    string `json:"instance_id" binding:"required"`
	Source        string `json:"source" binding:"required"`
	ProjectID     string `json:"project_id" binding:"required"`
	ProjectName   string `json:"project_name" binding:"required"`
	VersionID     string `json:"version_id" binding:"required"`
	VersionNumber string `json:"version_number"`
	Filename      string `json:"filename" binding:"required"`
	DownloadURL   string `json:"download_url" binding:"required"`
	ResourceType  string `json:"resource_type"`
	IconURL       string `json:"icon_url"`
	Downloads     int64  `json:"downloads"`
	FileSize      int64  `json:"file_size"`
}

type modInstallRequestResponse struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	InstanceID    string  `json:"instance_id"`
	Source        string  `json:"source"`
	ProjectID     string  `json:"project_id"`
	ProjectName   string  `json:"project_name"`
	VersionID     string  `json:"version_id"`
	VersionNumber string  `json:"version_number,omitempty"`
	Filename      string  `json:"filename"`
	DownloadURL   string  `json:"download_url,omitempty"`
	ResourceType  string  `json:"resource_type"`
	ErrorCode     *string `json:"error_code,omitempty"`
	ExpiresAt     string  `json:"expires_at"`
}

func modInstallResponseFromView(v *launcher.ModInstallRequestView, includeURL bool) modInstallRequestResponse {
	resp := modInstallRequestResponse{
		ID:            v.ID,
		Status:        v.Status,
		InstanceID:    v.InstanceID,
		Source:        v.Source,
		ProjectID:     v.ProjectID,
		ProjectName:   v.ProjectName,
		VersionID:     v.VersionID,
		VersionNumber: v.VersionNumber,
		Filename:      v.Filename,
		ResourceType:  v.ResourceType,
		ErrorCode:     v.ErrorCode,
		ExpiresAt:     v.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if includeURL {
		resp.DownloadURL = v.DownloadURL
	}
	return resp
}

func (h *ModInstallRequestsHandler) Create(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	deviceID, err := h.resolveDeviceID(c, owner)
	if err != nil {
		if errors.Is(err, launcher.ErrDeviceNotLinked) || errors.Is(err, launcher.ErrDeviceMismatch) {
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked to account")
			return
		}
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "device not found")
			return
		}
		JSONUnauthorized(c)
		return
	}

	var req createModInstallRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}

	view, err := h.Service.CreateModInstallRequest(c.Request.Context(), owner, launcher.CreateModInstallRequestInput{
		InstanceID:    req.InstanceID,
		DeviceID:      deviceID,
		Source:        req.Source,
		ProjectID:     req.ProjectID,
		ProjectName:   req.ProjectName,
		VersionID:     req.VersionID,
		VersionNumber: req.VersionNumber,
		Filename:      req.Filename,
		DownloadURL:   req.DownloadURL,
		ResourceType:  req.ResourceType,
		IconURL:       req.IconURL,
		Downloads:     req.Downloads,
		FileSize:      req.FileSize,
	})
	if err != nil {
		if mapModInstallServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusCreated, modInstallResponseFromView(view, false))
}

func (h *ModInstallRequestsHandler) Get(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.GetModInstallRequest(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "mod install request not found")
			return
		}
		if mapModInstallServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, modInstallResponseFromView(view, false))
}

func (h *ModInstallRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingModInstall(c.Request.Context(), deviceID)
	if err != nil {
		if mapModInstallServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": modInstallResponseFromView(view, true)})
}

type patchModInstallRequestBody struct {
	Status    string  `json:"status" binding:"required"`
	ErrorCode *string `json:"error_code"`
}

func (h *ModInstallRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchModInstallRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateModInstallRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdateModInstallRequestInput{
		Status:    req.Status,
		ErrorCode: req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "mod install request not found")
			return
		}
		if mapModInstallServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, modInstallResponseFromView(view, false))
}

func mapModInstallServiceError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, launcher.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return true
	case errors.Is(err, launcher.ErrValidation):
		JSONValidation(c, "invalid mod install request")
		return true
	case errors.Is(err, launcher.ErrInstanceManaged):
		JSONError(c, http.StatusForbidden, "INSTANCE_MANAGED", "this instance is managed by a game server")
		return true
	case errors.Is(err, launcher.ErrDeviceNotLinked), errors.Is(err, launcher.ErrDeviceMismatch):
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked to account")
		return true
	default:
		return false
	}
}

func (h *ModInstallRequestsHandler) resolveDeviceID(c *gin.Context, owner launcher.Owner) (string, error) {
	ctx := c.Request.Context()

	if header := strings.TrimSpace(c.GetHeader("X-Device-Token")); header != "" {
		token := strings.TrimPrefix(header, "Bearer ")
		return h.deviceIDFromToken(ctx, owner, token)
	}

	if authHeader := c.GetHeader("Authorization"); authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, err := h.Tokens.Parse(token); err == nil && claims.Kind == auth.TokenDevice && claims.DeviceID != "" {
			if err := h.Service.ValidateDeviceForOwner(ctx, owner, claims.DeviceID); err != nil {
				return "", err
			}
			return claims.DeviceID, nil
		}
	}

	deviceID, err := h.Service.FindLinkedDevice(ctx, owner)
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			return "", launcher.ErrDeviceNotLinked
		}
		return "", err
	}
	return deviceID, nil
}

func (h *ModInstallRequestsHandler) deviceIDFromToken(ctx context.Context, owner launcher.Owner, token string) (string, error) {
	claims, err := h.Tokens.Parse(token)
	if err != nil || claims.Kind != auth.TokenDevice || claims.DeviceID == "" {
		return "", launcher.ErrValidation
	}
	if err := h.Service.ValidateDeviceForOwner(ctx, owner, claims.DeviceID); err != nil {
		return "", err
	}
	return claims.DeviceID, nil
}
