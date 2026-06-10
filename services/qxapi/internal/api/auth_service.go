package api

import (
	"context"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type authService interface {
	Register(ctx context.Context, in auth.RegisterInput) (*models.User, *auth.TokenPair, error)
	Login(ctx context.Context, email, password string) (*models.User, *auth.TokenPair, error)
	GetUser(ctx context.Context, userID string) (*models.User, error)
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
	ChangeEmail(ctx context.Context, userID, currentPassword, newEmail string) (*models.User, error)
	Tokens() *auth.TokenService
}
