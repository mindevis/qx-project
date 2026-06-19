package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

func TestDeviceAuthMiddleware(t *testing.T) {
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	deviceToken, err := tokens.IssueDeviceToken("dev-1", time.Hour)
	if err != nil {
		t.Fatalf("device token: %v", err)
	}

	r := gin.New()
	r.GET("/ok", DeviceAuthMiddleware(tokens), func(c *gin.Context) {
		id, ok := deviceIDFromContext(c)
		if !ok || id != "dev-1" {
			t.Fatalf("device id: ok=%v id=%q", ok, id)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("X-Device-Token", deviceToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("x-device-token: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set("Authorization", "Bearer "+deviceToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("authorization bearer: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/ok", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth: %d", w.Code)
	}
}
