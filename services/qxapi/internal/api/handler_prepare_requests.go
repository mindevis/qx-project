package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

type PrepareRequestsHandler struct {
	Service *launcher.Service
}

type prepareRequestResponse struct {
	ID         string                      `json:"id"`
	Status     string                      `json:"status"`
	InstanceID string                      `json:"instance_id"`
	Instance   *launcher.LaunchInstanceView `json:"instance,omitempty"`
	ErrorCode  *string                     `json:"error_code,omitempty"`
	ExpiresAt  string                      `json:"expires_at"`
}

func prepareResponseFromView(v *launcher.PrepareRequestView, includeInstance bool) prepareRequestResponse {
	resp := prepareRequestResponse{
		ID:         v.ID,
		Status:     v.Status,
		InstanceID: v.InstanceID,
		ErrorCode:  v.ErrorCode,
		ExpiresAt:  v.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if includeInstance {
		resp.Instance = v.Instance
	}
	return resp
}

func (h *PrepareRequestsHandler) Get(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.GetPrepareRequest(c.Request.Context(), owner, c.Param("id"))
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "prepare request not found")
			return
		}
		if mapPrepareServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, prepareResponseFromView(view, false))
}

func (h *PrepareRequestsHandler) Pending(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	view, err := h.Service.FetchPendingPrepare(c.Request.Context(), deviceID)
	if err != nil {
		if mapPrepareServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	if view == nil {
		c.JSON(http.StatusOK, gin.H{"item": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": prepareResponseFromView(view, true)})
}

type patchPrepareRequestBody struct {
	Status    string  `json:"status" binding:"required"`
	ErrorCode *string `json:"error_code"`
}

func (h *PrepareRequestsHandler) Update(c *gin.Context) {
	deviceID, ok := deviceIDFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req patchPrepareRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	view, err := h.Service.UpdatePrepareRequest(c.Request.Context(), deviceID, c.Param("id"), launcher.UpdatePrepareRequestInput{
		Status:    req.Status,
		ErrorCode: req.ErrorCode,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "prepare request not found")
			return
		}
		if mapPrepareServiceError(c, err) {
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, prepareResponseFromView(view, false))
}

func mapPrepareServiceError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, launcher.ErrNotFound):
		JSONError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return true
	case errors.Is(err, launcher.ErrValidation):
		JSONValidation(c, "invalid prepare request")
		return true
	default:
		return false
	}
}
