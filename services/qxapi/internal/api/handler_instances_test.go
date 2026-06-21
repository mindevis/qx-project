package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func newInstancesHandler(t *testing.T) (*InstancesHandler, *auth.TokenService) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	svc.SetManifestProvider(testManifestProvider{})
	return &InstancesHandler{Service: svc}, tokens
}

func TestInstancesHandlerCRUD(t *testing.T) {
	h, tokens := newInstancesHandler(t)
	pair, err := tokens.IssueUserTokens("user-1", "u@test.com")
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}
	claims, _ := tokens.Parse(pair.AccessToken)

	body, _ := json.Marshal(map[string]string{
		"name":       "Test",
		"mc_version": "1.21",
		"loader":     models.LoaderVanilla,
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, claims.UserID)
	h.Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	var created instanceResponse
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, claims.UserID)
	h.List(c)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, claims.UserID)
	h.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, claims.UserID)
	h.Delete(c)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", w.Code)
	}
}

func TestInstancesHandlerGuestForbiddenLoader(t *testing.T) {
	h, tokens := newInstancesHandler(t)
	guestToken, _, err := tokens.IssueGuestToken("guest-1")
	if err != nil {
		t.Fatalf("guest token: %v", err)
	}
	claims, _ := tokens.Parse(guestToken)

	body, _ := json.Marshal(map[string]string{
		"name":       "Modded",
		"mc_version": "1.21",
		"loader":     "fabric",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(GuestSessionIDKey, claims.UserID)
	c.Set(IsGuestKey, true)
	h.Create(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("forbidden: %d", w.Code)
	}
}

func TestInstancesHandlerUnauthorized(t *testing.T) {
	h, _ := newInstancesHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h.List(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized: %d", w.Code)
	}
}

func TestInstancesHandlerGetNotFound(t *testing.T) {
	h, tokens := newInstancesHandler(t)
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000099"}}
	c.Set(UserIDKey, claims.UserID)
	h.Get(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}
}

func TestInstancesHandlerDeleteNotFound(t *testing.T) {
	h, tokens := newInstancesHandler(t)
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000099"}}
	c.Set(UserIDKey, claims.UserID)
	h.Delete(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}
}

func TestInstancesHandlerManifest(t *testing.T) {
	h, tokens := newInstancesHandler(t)
	ctx := context.Background()
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)
	owner := launcher.Owner{UserID: claims.UserID, IsGuest: false}

	inst, err := h.Service.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: inst.ID}}
	c.Set(UserIDKey, claims.UserID)
	h.Manifest(c)
	if w.Code != http.StatusOK {
		t.Fatalf("manifest: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000099"}}
	c.Set(UserIDKey, claims.UserID)
	h.Manifest(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("manifest not found: %d", w.Code)
	}
}
