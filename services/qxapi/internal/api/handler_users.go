package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type UsersHandler struct {
	Service authService
}

type profileResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Username  *string `json:"username,omitempty"`
	Tier      string  `json:"tier"`
	CreatedAt string  `json:"created_at"`
}

func (h *UsersHandler) Me(c *gin.Context) {
	userID, _ := c.Get(UserIDKey)
	user, err := h.Service.GetUser(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		JSONInternal(c)
		return
	}
	c.JSON(http.StatusOK, profileFromUser(user))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type changeEmailRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
}

func (h *UsersHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}

	userID, _ := c.Get(UserIDKey)
	err := h.Service.ChangePassword(c.Request.Context(), userID.(string), req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrWrongPassword):
			JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current password is incorrect")
		case errors.Is(err, auth.ErrValidation):
			JSONValidation(c, "invalid password data")
		default:
			if errors.Is(err, gorm.ErrRecordNotFound) {
				JSONError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
				return
			}
			JSONInternal(c)
		}
		return
	}

	c.AbortWithStatus(http.StatusNoContent)
}

func (h *UsersHandler) ChangeEmail(c *gin.Context) {
	var req changeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		JSONValidation(c, err.Error())
		return
	}

	userID, _ := c.Get(UserIDKey)
	user, err := h.Service.ChangeEmail(c.Request.Context(), userID.(string), req.CurrentPassword, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrWrongPassword):
			JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current password is incorrect")
		case errors.Is(err, auth.ErrEmailTaken):
			JSONConflict(c, "email already registered")
		case errors.Is(err, auth.ErrValidation):
			JSONValidation(c, "invalid email data")
		default:
			if errors.Is(err, gorm.ErrRecordNotFound) {
				JSONError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
				return
			}
			JSONInternal(c)
		}
		return
	}

	c.JSON(http.StatusOK, profileFromUser(user))
}

func profileFromUser(user *models.User) profileResponse {
	return profileResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Tier:      user.Tier,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	}
}
