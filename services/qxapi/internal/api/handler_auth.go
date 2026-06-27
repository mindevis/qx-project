package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

type AuthHandler struct {
	Service authService
}

type registerRequest struct {
	Email    string  `json:"email" binding:"required,email"`
	Password string  `json:"password" binding:"required,min=8"`
	Username *string `json:"username"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

func tokenFromPair(pair *auth.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	_, pair, err := h.Service.Register(c.Request.Context(), auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailTaken):
			JSONConflict(c, "email already registered")
		case errors.Is(err, auth.ErrValidation):
			JSONValidation(c, "invalid registration data")
		default:
			JSONInternal(c)
		}
		return
	}
	c.JSON(http.StatusCreated, tokenFromPair(pair))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	_, pair, err := h.Service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidLogin) {
			JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, tokenFromPair(pair))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}
	pair, err := h.Service.Tokens().Refresh(req.RefreshToken)
	if err != nil {
		JSONUnauthorized(c)
		return
	}
	c.JSON(http.StatusOK, tokenFromPair(pair))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.AbortWithStatus(http.StatusNoContent)
}
