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

type LaunchRequestsHandler struct {
	Service *launcher.Service
	Tokens  *auth.TokenService
}

type createLaunchRequestBody struct {
	InstanceID        string `json:"instance_id" binding:"required"`
	OfflineProfileID  string `json:"offline_profile_id"`
	UseMojangAccount  bool   `json:"use_mojang_account"`
	JoinServerAddress string `json:"join_server_address"`
	JoinServerPort    int    `json:"join_server_port"`
}

type launchRequestResponse struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	InstanceID       string `json:"instance_id"`
	OfflineProfileID *string `json:"offline_profile_id,omitempty"`
	ExpiresAt        string `json:"expires_at"`
	PID              *int   `json:"pid,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	ErrorCode        *string `json:"error_code,omitempty"`
}

func launchResponseFromView(v *launcher.LaunchRequestView) launchRequestResponse {
	return launchRequestResponse{
		ID:               v.ID,
		Status:           v.Status,
		InstanceID:       v.InstanceID,
		OfflineProfileID: v.OfflineProfileID,
		ExpiresAt:        v.ExpiresAt.UTC().Format(time.RFC3339),
		PID:              v.PID,
		ExitCode:         v.ExitCode,
		ErrorCode:        v.ErrorCode,
	}
}

func (h *LaunchRequestsHandler) Create(c *gin.Context) {
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

	var req createLaunchRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}

	view, err := h.Service.CreateLaunchRequest(c.Request.Context(), owner, launcher.CreateLaunchRequestInput{
		InstanceID:        req.InstanceID,
		OfflineProfileID:  req.OfflineProfileID,
		DeviceID:          deviceID,
		UseMojangAccount:  req.UseMojangAccount,
		JoinServerAddress: req.JoinServerAddress,
		JoinServerPort:    req.JoinServerPort,
	})
	if err != nil {
		if mapLaunchServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusCreated, launchResponseFromView(view))
}

func mapLaunchServiceError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, launcher.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return true
	case errors.Is(err, launcher.ErrValidation):
		JSONValidation(c, "invalid launch request")
		return true
	case errors.Is(err, launcher.ErrDeviceNotLinked), errors.Is(err, launcher.ErrDeviceMismatch):
		JSONError(c, http.StatusForbidden, "FORBIDDEN", "device not linked to account")
		return true
	case errors.Is(err, launcher.ErrManifest):
		JSONError(c, http.StatusBadGateway, "MANIFEST_UNAVAILABLE", "could not build launch manifest")
		return true
	case errors.Is(err, launcher.ErrMojangSession):
		JSONError(c, http.StatusUnauthorized, "MOJANG_SESSION", "mojang session expired or unavailable")
		return true
	case errors.Is(err, launcher.ErrMojangUnavailable):
		JSONError(c, http.StatusBadGateway, "MOJANG_UNAVAILABLE", "microsoft authentication is temporarily unavailable")
		return true
	default:
		return false
	}
}

func (h *LaunchRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingLaunch(c.Request.Context(), deviceID)
	if err != nil {
		if mapLaunchServiceError(c, err) {
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

type patchLaunchRequestBody struct {
	Status    string  `json:"status" binding:"required"`
	PID       *int    `json:"pid"`
	ExitCode  *int    `json:"exit_code"`
	ErrorCode *string `json:"error_code"`
}

func (h *LaunchRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchLaunchRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdateLaunchRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdateLaunchRequestInput{
		Status:    req.Status,
		PID:       req.PID,
		ExitCode:  req.ExitCode,
		ErrorCode: req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "launch request not found")
			return
		}
		if mapLaunchServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, launchResponseFromView(view))
}

func (h *LaunchRequestsHandler) Get(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.GetLaunchRequest(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "launch request not found")
			return
		}
		if mapLaunchServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, launchResponseFromView(view))
}

func (h *LaunchRequestsHandler) resolveDeviceID(c *gin.Context, owner launcher.Owner) (string, error) {
	ctx := c.Request.Context()

	if header := strings.TrimSpace(c.GetHeader("X-Device-Token")); header != "" {
		token := strings.TrimPrefix(header, "Bearer ")
		deviceID, err := h.deviceIDFromToken(ctx, owner, token)
		if err != nil {
			return "", err
		}
		return deviceID, nil
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

func (h *LaunchRequestsHandler) deviceIDFromToken(ctx context.Context, owner launcher.Owner, token string) (string, error) {
	claims, err := h.Tokens.Parse(token)
	if err != nil || claims.Kind != auth.TokenDevice || claims.DeviceID == "" {
		return "", launcher.ErrValidation
	}
	if err := h.Service.ValidateDeviceForOwner(ctx, owner, claims.DeviceID); err != nil {
		return "", err
	}
	return claims.DeviceID, nil
}
