package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var (
	ErrEmailTaken   = errors.New("email already registered")
	ErrInvalidLogin = errors.New("invalid email or password")
	ErrWrongPassword = errors.New("current password is incorrect")
	ErrValidation   = errors.New("validation error")
)

type tokenIssuer interface {
	IssueUserTokens(userID, email string) (*TokenPair, error)
}

type Service struct {
	db     *gorm.DB
	tokens tokenIssuer
}

func NewService(db *gorm.DB, tokens tokenIssuer) *Service {
	return &Service{db: db, tokens: tokens}
}

type RegisterInput struct {
	Email    string
	Password string
	Username *string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*models.User, *TokenPair, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" || len(in.Password) < 8 {
		return nil, nil, ErrValidation
	}

	var existing models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&existing).Error; err == nil {
		return nil, nil, ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, nil, err
	}

	user := models.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		Username:     in.Username,
		Tier:         "free",
	}
	if err := createUser(s.db.WithContext(ctx), &user); err != nil {
		return nil, nil, err
	}

	pair, err := s.tokens.IssueUserTokens(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}
	return &user, pair, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*models.User, *TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidLogin
		}
		return nil, nil, err
	}
	if !CheckPassword(user.PasswordHash, password) {
		return nil, nil, ErrInvalidLogin
	}
	pair, err := s.tokens.IssueUserTokens(user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}
	return &user, pair, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrValidation
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(user.PasswordHash, currentPassword) {
		return ErrWrongPassword
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("password_hash", hash).Error
}

func (s *Service) ChangeEmail(ctx context.Context, userID, currentPassword, newEmail string) (*models.User, error) {
	newEmail = strings.TrimSpace(strings.ToLower(newEmail))
	if newEmail == "" {
		return nil, ErrValidation
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !CheckPassword(user.PasswordHash, currentPassword) {
		return nil, ErrWrongPassword
	}
	if newEmail == user.Email {
		return user, nil
	}

	if err := checkEmailAvailable(s.db, ctx, newEmail); err != nil {
		return nil, err
	}

	if err := updateUserEmail(s.db, ctx, userID, newEmail); err != nil {
		return nil, err
	}

	return reloadUserAfterEmailChange(s, ctx, userID)
}

var reloadUserAfterEmailChange = func(s *Service, ctx context.Context, userID string) (*models.User, error) {
	return s.GetUser(ctx, userID)
}

var createUser = func(db *gorm.DB, user *models.User) error {
	return db.Create(user).Error
}

var checkEmailAvailable = func(db *gorm.DB, ctx context.Context, email string) error {
	var existing models.User
	if err := db.WithContext(ctx).Where("email = ?", email).First(&existing).Error; err == nil {
		return ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

var updateUserEmail = func(db *gorm.DB, ctx context.Context, userID, email string) error {
	return db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("email", email).Error
}

func (s *Service) Tokens() *TokenService {
	if ts, ok := s.tokens.(*TokenService); ok {
		return ts
	}
	return nil
}
