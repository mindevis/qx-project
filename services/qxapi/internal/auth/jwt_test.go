package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestBreakSigningForTest(t *testing.T) {
	BreakSigningForTest(t)
	svc := NewTokenService("secret", time.Minute, time.Hour)
	if _, err := svc.IssueUserTokens("u", "e@e.com"); err == nil {
		t.Fatal("expected signing to fail")
	}
}

func TestTokenServiceUserTokens(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	pair, err := svc.IssueUserTokens("user-1", "a@b.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
	if pair.ExpiresIn != 60 {
		t.Fatalf("expires_in: got %d", pair.ExpiresIn)
	}

	access, err := svc.Parse(pair.AccessToken)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if access.Kind != TokenAccess || access.UserID != "user-1" || access.Email != "a@b.com" {
		t.Fatalf("unexpected claims: %+v", access)
	}

	refreshed, err := svc.Refresh(pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.AccessToken == "" {
		t.Fatal("expected new access token")
	}
}

func TestTokenServiceGuestToken(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	token, ttl, err := svc.IssueGuestToken("device-1")
	if err != nil {
		t.Fatalf("guest: %v", err)
	}
	if token == "" || ttl != 24*time.Hour {
		t.Fatalf("token=%q ttl=%v", token, ttl)
	}
	claims, err := svc.Parse(token)
	if err != nil || claims.Kind != TokenGuest {
		t.Fatalf("parse guest: %v claims=%+v", err, claims)
	}
}

func TestTokenServiceParseErrors(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)

	if _, err := svc.Parse("not-a-jwt"); err == nil {
		t.Fatal("expected parse error")
	}

	pair, _ := svc.IssueUserTokens("u", "e@e.com")
	if _, err := svc.Refresh(pair.AccessToken); err == nil {
		t.Fatal("expected refresh with access token to fail")
	}

	other := NewTokenService("other", time.Minute, time.Hour)
	if _, err := other.Parse(pair.AccessToken); err == nil {
		t.Fatal("expected wrong secret to fail")
	}
}

func TestTokenServiceSignErrors(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	old := signToken
	signToken = func(s *TokenService, kind TokenKind, userID, email string, ttl time.Duration) (string, error) {
		if kind == TokenAccess {
			return "access", nil
		}
		return "", errors.New("sign failed")
	}
	t.Cleanup(func() { signToken = old })

	if _, err := svc.IssueUserTokens("u", "e@e.com"); err == nil {
		t.Fatal("expected refresh sign error")
	}

	signToken = func(*TokenService, TokenKind, string, string, time.Duration) (string, error) {
		return "", errors.New("sign failed")
	}
	if _, err := svc.IssueUserTokens("u", "e@e.com"); err == nil {
		t.Fatal("expected access sign error")
	}
	if _, _, err := svc.IssueGuestToken("d"); err == nil {
		t.Fatal("expected guest sign error")
	}
}

func TestTokenServiceExpiredToken(t *testing.T) {
	svc := NewTokenService("secret", time.Millisecond, time.Hour)
	pair, err := svc.IssueUserTokens("u", "e@e.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := svc.Parse(pair.AccessToken); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestTokenServiceRefreshParseError(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	if _, err := svc.Refresh("not-a-jwt"); err == nil {
		t.Fatal("expected refresh parse error")
	}
}

func TestTokenServiceRefreshIssueError(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	pair, err := svc.IssueUserTokens("u", "e@e.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	old := signToken
	signToken = func(*TokenService, TokenKind, string, string, time.Duration) (string, error) {
		return "", errors.New("sign failed")
	}
	t.Cleanup(func() { signToken = old })

	if _, err := svc.Refresh(pair.RefreshToken); err == nil {
		t.Fatal("expected refresh issue error")
	}
}

func TestTokenServiceParseInvalidToken(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	pair, err := svc.IssueUserTokens("u", "e@e.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	old := parseWithClaimsFn
	parseWithClaimsFn = func(string, jwt.Claims, jwt.Keyfunc, ...jwt.ParserOption) (*jwt.Token, error) {
		return &jwt.Token{Valid: false, Claims: &Claims{Kind: TokenAccess}}, nil
	}
	t.Cleanup(func() { parseWithClaimsFn = old })

	if _, err := svc.Parse(pair.AccessToken); err == nil {
		t.Fatal("expected invalid token error")
	}
}

func TestTokenServiceParseInvalidClaimsType(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	pair, err := svc.IssueUserTokens("u", "e@e.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	old := newParseClaims
	newParseClaims = func() jwt.Claims { return jwt.MapClaims{} }
	t.Cleanup(func() { newParseClaims = old })

	if _, err := svc.Parse(pair.AccessToken); err == nil {
		t.Fatal("expected invalid claims type error")
	}
}

func TestTokenServiceUnexpectedSigningMethod(t *testing.T) {
	svc := NewTokenService("secret", time.Minute, time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, &Claims{
		Kind:   TokenAccess,
		UserID: "x",
	})
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := svc.Parse(signed); err == nil {
		t.Fatal("expected HS512 token to be rejected")
	}
}
