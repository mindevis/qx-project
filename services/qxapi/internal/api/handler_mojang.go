package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/mojang"
)

type MojangHandler struct {
	Service *mojang.Service
}

func (h *MojangHandler) Status(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	status, err := h.Service.GetStatus(c.Request.Context(), owner.UserID)
	if err != nil {
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *MojangHandler) StartOAuth(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	url, err := h.Service.BeginOAuth(c.Request.Context(), owner.UserID)
	if err != nil {
		if errors.Is(err, mojang.ErrNotConfigured) {
			JSONError(c, http.StatusServiceUnavailable, "MOJANG_NOT_CONFIGURED", "microsoft oauth is not configured")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"authorization_url": url})
}

func (h *MojangHandler) OAuthCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		JSONValidation(c, "missing code or state")
		return
	}
	redirectURL, err := h.Service.CompleteOAuth(c.Request.Context(), code, state)
	if err != nil {
		if errors.Is(err, mojang.ErrInvalidOAuth) {
			JSONError(c, http.StatusBadRequest, "INVALID_OAUTH_STATE", "oauth state expired or invalid")
			return
		}
		if errors.Is(err, mojang.ErrAlreadyLinked) {
			JSONError(c, http.StatusConflict, "MOJANG_ALREADY_LINKED", "this minecraft account is linked to another qx user")
			return
		}
		JSONError(c, http.StatusBadGateway, "MOJANG_OAUTH_FAILED", err.Error())
		return
	}
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *MojangHandler) Unlink(c *gin.Context) {
	owner, ok := ownerFromContext(c)
	if !ok {
		JSONUnauthorized(c)
		return
	}
	if err := h.Service.Unlink(c.Request.Context(), owner.UserID); err != nil {
		JSONInternal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
