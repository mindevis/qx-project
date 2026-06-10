package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenKind string

const (
	TokenAccess  TokenKind = "access"
	TokenRefresh TokenKind = "refresh"
	TokenGuest   TokenKind = "guest"
)

type Claims struct {
	jwt.RegisteredClaims
	Kind   TokenKind `json:"kind"`
	UserID string    `json:"user_id,omitempty"`
	Email  string    `json:"email,omitempty"`
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type TokenService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration) *TokenService {
	return &TokenService{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

func (s *TokenService) IssueUserTokens(userID, email string) (*TokenPair, error) {
	access, err := signToken(s, TokenAccess, userID, email, s.accessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := signToken(s, TokenRefresh, userID, email, s.refreshTokenTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *TokenService) IssueGuestToken(guestSessionID string) (string, time.Duration, error) {
	token, err := signToken(s, TokenGuest, guestSessionID, "", 24*time.Hour)
	if err != nil {
		return "", 0, err
	}
	return token, 24 * time.Hour, nil
}

var newParseClaims = func() jwt.Claims { return &Claims{} }

var parseWithClaimsFn = jwt.ParseWithClaims

func (s *TokenService) Parse(token string) (*Claims, error) {
	parsed, err := parseWithClaimsFn(token, newParseClaims(), func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *TokenService) Refresh(refreshToken string) (*TokenPair, error) {
	claims, err := s.Parse(refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.Kind != TokenRefresh {
		return nil, fmt.Errorf("not a refresh token")
	}
	return s.IssueUserTokens(claims.UserID, claims.Email)
}

var signToken = func(s *TokenService, kind TokenKind, userID, email string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Kind:   kind,
		UserID: userID,
		Email:  email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// BreakSigningForTest stubs JWT signing until t completes.
func BreakSigningForTest(t interface{ Cleanup(func()) }) {
	old := signToken
	signToken = func(*TokenService, TokenKind, string, string, time.Duration) (string, error) {
		return "", fmt.Errorf("sign failed")
	}
	t.Cleanup(func() { signToken = old })
}
