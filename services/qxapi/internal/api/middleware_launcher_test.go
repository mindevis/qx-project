package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

func TestLauncherOwnerMiddleware(t *testing.T) {
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	pair, err := tokens.IssueUserTokens("user-1", "u@test.com")
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}

	r := gin.New()
	r.GET("/ok", LauncherOwnerMiddleware(tokens), func(c *gin.Context) {
		if _, ok := ownerFromContext(c); !ok {
			t.Fatal("expected owner")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("user auth: %d", w.Code)
	}

	guestToken, _, err := tokens.IssueGuestToken("guest-1")
	if err != nil {
		t.Fatalf("guest: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Authorization", "Bearer "+guestToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("guest auth: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/ok", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth: %d", w.Code)
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	r := gin.New()
	r.GET("/opt", OptionalAuthMiddleware(tokens), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/opt", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("no auth: %d", w.Code)
	}
}
