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
	TokenDevice  TokenKind = "device"
	TokenAgent   TokenKind = "agent"
)

type Claims struct {
	jwt.RegisteredClaims
	Kind           TokenKind `json:"kind"`
	UserID         string    `json:"user_id,omitempty"`
	Email          string    `json:"email,omitempty"`
	DeviceID       string    `json:"device_id,omitempty"`
	GuestSessionID string    `json:"guest_session_id,omitempty"`
	ServerID       string    `json:"server_id,omitempty"`
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
	token, err := signGuestToken(s, guestSessionID, 24*time.Hour)
	if err != nil {
		return "", 0, err
	}
	return token, 24 * time.Hour, nil
}

func (s *TokenService) IssueDeviceToken(deviceID string, ttl time.Duration) (string, error) {
	return signDeviceToken(s, deviceID, ttl)
}

func (s *TokenService) IssueAgentToken(serverID string, ttl time.Duration) (string, error) {
	return signAgentToken(s, serverID, ttl)
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

var signGuestToken = func(s *TokenService, guestSessionID string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   guestSessionID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Kind:           TokenGuest,
		UserID:         guestSessionID,
		GuestSessionID: guestSessionID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

var signDeviceToken = func(s *TokenService, deviceID string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   deviceID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Kind:     TokenDevice,
		DeviceID: deviceID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

var signAgentToken = func(s *TokenService, serverID string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "agent:" + serverID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		Kind:     TokenAgent,
		ServerID: serverID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
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

	oldGuest := signGuestToken
	signGuestToken = func(*TokenService, string, time.Duration) (string, error) {
		return "", fmt.Errorf("sign failed")
	}
	t.Cleanup(func() { signGuestToken = oldGuest })

	oldDevice := signDeviceToken
	signDeviceToken = func(*TokenService, string, time.Duration) (string, error) {
		return "", fmt.Errorf("sign failed")
	}
	t.Cleanup(func() { signDeviceToken = oldDevice })

	oldAgent := signAgentToken
	signAgentToken = func(*TokenService, string, time.Duration) (string, error) {
		return "", fmt.Errorf("sign failed")
	}
	t.Cleanup(func() { signAgentToken = oldAgent })
}
