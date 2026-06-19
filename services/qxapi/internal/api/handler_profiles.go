package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type ProfilesHandler struct {
	Service *launcher.Service
}

type offlineProfileResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	OfflineUUID string `json:"offline_uuid"`
	CreatedAt   string `json:"created_at"`
}

func offlineProfileFromModel(p models.OfflineProfile) offlineProfileResponse {
	return offlineProfileResponse{
		ID:          p.ID,
		Username:    p.Username,
		OfflineUUID: p.OfflineUUID,
		CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type createProfileRequest struct {
	Username string `json:"username" binding:"required"`
}

func (h *ProfilesHandler) List(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	items, err := h.Service.ListProfiles(c.Request.Context(), owner)
	if err != nil {
		JSONInternal(c)
		return
	}
	out := make([]offlineProfileResponse, 0, len(items))
	for _, item := range items {
		out = append(out, offlineProfileFromModel(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *ProfilesHandler) Create(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	var req createProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	profile, err := h.Service.CreateProfile(c.Request.Context(), owner, launcher.CreateProfileInput{
		Username: req.Username,
	})
	if err != nil {
		if errors.Is(err, launcher.ErrValidation) {
			JSONValidation(c, "invalid username (3-16 chars, a-z A-Z 0-9 _)")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusCreated, offlineProfileFromModel(*profile))
}

func (h *ProfilesHandler) Delete(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if err := h.Service.DeleteProfile(c.Request.Context(), owner, c.Param("id")); err != nil {
		if errors.Is(err, launcher.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}
