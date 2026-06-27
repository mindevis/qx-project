package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")

	t.Run("missing header", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthMiddleware(tokens))
		r.GET("/", func(c *gin.Context) { c.Status(200) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status: %d", w.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthMiddleware(tokens))
		r.GET("/", func(c *gin.Context) { c.Status(200) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status: %d", w.Code)
		}
	})

	t.Run("refresh token rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(AuthMiddleware(tokens))
		r.GET("/", func(c *gin.Context) { c.Status(200) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status: %d", w.Code)
		}
	})

	t.Run("valid access", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		var gotID string
		r.Use(AuthMiddleware(tokens))
		r.GET("/", func(c *gin.Context) {
			v, _ := c.Get(UserIDKey)
			gotID, _ = v.(string)
			c.Status(200)
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		r.ServeHTTP(w, req)
		if w.Code != 200 || gotID != "user-1" {
			t.Fatalf("status=%d id=%v", w.Code, gotID)
		}
	})
}

func TestCORSMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(CORSMiddleware("http://localhost:5173"))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	t.Run("preflight", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		r.ServeHTTP(w, req)
		if w.Code != 204 {
			t.Fatalf("status: %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
			t.Fatal("missing cors header")
		}
	})

	t.Run("get", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status: %d", w.Code)
		}
	})
}

func TestDeviceOrLauncherOwnerMiddleware(t *testing.T) {
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	deviceTok, _ := tokens.IssueDeviceToken("dev-1", time.Hour)

	t.Run("device token", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		var gotID string
		r.Use(DeviceOrLauncherOwnerMiddleware(tokens))
		r.POST("/", func(c *gin.Context) {
			id, ok := deviceIDFromContext(c)
			if ok {
				gotID = id
			}
			c.Status(200)
		})
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer "+deviceTok)
		r.ServeHTTP(w, req)
		if w.Code != 200 || gotID != "dev-1" {
			t.Fatalf("status=%d id=%q", w.Code, gotID)
		}
	})

	t.Run("user token", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		var gotID string
		r.Use(DeviceOrLauncherOwnerMiddleware(tokens))
		r.POST("/", func(c *gin.Context) {
			owner, ok := ownerFromContext(c)
			if ok {
				gotID = owner.UserID
			}
			c.Status(200)
		})
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		r.ServeHTTP(w, req)
		if w.Code != 200 || gotID != "user-1" {
			t.Fatalf("status=%d id=%q", w.Code, gotID)
		}
	})

	t.Run("missing auth", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(DeviceOrLauncherOwnerMiddleware(tokens))
		r.POST("/", func(c *gin.Context) { c.Status(200) })
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status: %d", w.Code)
		}
	})

	t.Run("invalid bearer", func(t *testing.T) {
		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(DeviceOrLauncherOwnerMiddleware(tokens))
		r.POST("/", func(c *gin.Context) { c.Status(200) })
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer not-a-jwt")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status: %d", w.Code)
		}
	})
}
