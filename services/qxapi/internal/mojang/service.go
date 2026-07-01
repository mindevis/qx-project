package mojang

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/msauth"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var (
	ErrNotLinked       = errors.New("mojang account not linked")
	ErrNotConfigured   = errors.New("mojang oauth not configured")
	ErrInvalidOAuth    = errors.New("invalid oauth state")
	ErrAlreadyLinked   = errors.New("mojang account already linked to another user")
	ErrOAuthInProgress = errors.New("oauth flow failed")
)

const oauthStateTTL = 10 * time.Minute

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	WebBaseURL   string
}

type LinkStatus struct {
	Linked        bool      `json:"linked"`
	Username      string    `json:"username,omitempty"`
	MinecraftUUID string    `json:"minecraft_uuid,omitempty"`
	LinkedAt      time.Time `json:"linked_at,omitempty"`
}

type SessionView struct {
	Username    string `json:"username"`
	UUID        string `json:"uuid"`
	AccessToken string `json:"access_token"`
}

type Service struct {
	db     *gorm.DB
	enc    *crypto.Encryptor
	ms     *msauth.Client
	secret []byte
	cfg    Config
}

func NewService(db *gorm.DB, enc *crypto.Encryptor, cfg Config, jwtSecret string) *Service {
	msCfg := msauth.Config{
		ClientID:     strings.TrimSpace(cfg.ClientID),
		ClientSecret: strings.TrimSpace(cfg.ClientSecret),
		RedirectURI:  strings.TrimSpace(cfg.RedirectURI),
	}
	return &Service{
		db:     db,
		enc:    enc,
		ms:     msauth.NewClient(msCfg),
		secret: []byte(jwtSecret),
		cfg:    cfg,
	}
}

type oauthStateClaims struct {
	UserID   string `json:"uid"`
	Verifier string `json:"ver"`
	jwt.RegisteredClaims
}

func (s *Service) configured() bool {
	return strings.TrimSpace(s.cfg.RedirectURI) != "" && len(s.secret) > 0
}

func (s *Service) GetStatus(ctx context.Context, userID string) (*LinkStatus, error) {
	var link models.MojangLink
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &LinkStatus{Linked: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &LinkStatus{
		Linked:        true,
		Username:      link.Username,
		MinecraftUUID: link.MinecraftUUID,
		LinkedAt:      link.LinkedAt,
	}, nil
}

func (s *Service) BeginOAuth(ctx context.Context, userID string) (string, error) {
	if !s.configured() {
		return "", ErrNotConfigured
	}
	begin, err := s.ms.BeginAuthorize()
	if err != nil {
		return "", err
	}
	stateToken, err := s.signOAuthState(userID, begin.Verifier)
	if err != nil {
		return "", err
	}
	return replaceOAuthState(begin.URL, stateToken), nil
}

func (s *Service) CompleteOAuth(ctx context.Context, code, stateToken string) (string, error) {
	if !s.configured() {
		return "", ErrNotConfigured
	}
	userID, verifier, err := s.parseOAuthState(stateToken)
	if err != nil {
		return "", err
	}
	tokens, err := s.ms.ExchangeCode(ctx, code, verifier)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthInProgress, err)
	}
	session, err := s.ms.Login(ctx, tokens.AccessToken)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrOAuthInProgress, err)
	}
	if err := s.saveLink(ctx, userID, session, tokens.RefreshToken); err != nil {
		return "", err
	}
	return strings.TrimRight(s.cfg.WebBaseURL, "/") + "/profile?mojang=linked", nil
}

func (s *Service) Unlink(ctx context.Context, userID string) error {
	res := s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.MojangLink{})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (s *Service) SessionForLaunch(ctx context.Context, userID string) (*SessionView, error) {
	var link models.MojangLink
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotLinked
		}
		return nil, err
	}
	refresh, err := s.decryptString(link.RefreshTokenEnc)
	if err != nil {
		return nil, err
	}
	session, tokens, err := s.ms.LoginFromRefresh(ctx, refresh)
	if err != nil {
		return nil, ClassifyAuthError(err)
	}
	if tokens.RefreshToken != "" && tokens.RefreshToken != refresh {
		enc, err := s.encryptString(tokens.RefreshToken)
		if err != nil {
			return nil, err
		}
		_ = s.db.WithContext(ctx).Model(&link).Update("refresh_token_enc", enc).Error
	}
	if session.Username != link.Username || session.UUID != link.MinecraftUUID {
		_ = s.db.WithContext(ctx).Model(&link).Updates(map[string]any{
			"username":       session.Username,
			"minecraft_uuid": msauth.GameUUID(session.UUID),
		}).Error
	}
	return &SessionView{
		Username:    session.Username,
		UUID:        msauth.GameUUID(session.UUID),
		AccessToken: session.AccessToken,
	}, nil
}

func (s *Service) saveLink(ctx context.Context, userID string, session *msauth.Session, refreshToken string) error {
	mcUUID := msauth.GameUUID(session.UUID)
	var existing models.MojangLink
	err := s.db.WithContext(ctx).Where("minecraft_uuid = ?", mcUUID).First(&existing).Error
	if err == nil && existing.UserID != userID {
		return ErrAlreadyLinked
	}
	refreshEnc, err := s.encryptString(refreshToken)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	link := models.MojangLink{
		UserID:          userID,
		MinecraftUUID:   mcUUID,
		Username:        session.Username,
		RefreshTokenEnc: refreshEnc,
		LinkedAt:        now,
	}
	return s.db.WithContext(ctx).Save(&link).Error
}

func (s *Service) signOAuthState(userID, verifier string) (string, error) {
	now := time.Now().UTC()
	claims := oauthStateClaims{
		UserID:   userID,
		Verifier: verifier,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(oauthStateTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *Service) parseOAuthState(stateToken string) (userID, verifier string, err error) {
	parsed, err := jwt.ParseWithClaims(stateToken, &oauthStateClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return "", "", ErrInvalidOAuth
	}
	claims, ok := parsed.Claims.(*oauthStateClaims)
	if !ok || !parsed.Valid || claims.UserID == "" || claims.Verifier == "" {
		return "", "", ErrInvalidOAuth
	}
	return claims.UserID, claims.Verifier, nil
}

func replaceOAuthState(authorizeURL, stateToken string) string {
	parts := strings.SplitN(authorizeURL, "?", 2)
	if len(parts) != 2 {
		return authorizeURL
	}
	values := strings.Split(parts[1], "&")
	out := make([]string, 0, len(values))
	for _, pair := range values {
		if strings.HasPrefix(pair, "state=") {
			out = append(out, "state="+stateToken)
			continue
		}
		out = append(out, pair)
	}
	return parts[0] + "?" + strings.Join(out, "&")
}

func (s *Service) encryptString(value string) ([]byte, error) {
	return s.enc.Encrypt([]byte(value))
}

func (s *Service) decryptString(value []byte) (string, error) {
	plain, err := s.enc.Decrypt(value)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
