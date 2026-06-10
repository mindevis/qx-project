package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

type mockAuthService struct {
	regErr   error
	loginErr error
	inner    *auth.Service
}

func (m *mockAuthService) Register(ctx context.Context, in auth.RegisterInput) (*models.User, *auth.TokenPair, error) {
	if m.regErr != nil {
		return nil, nil, m.regErr
	}
	return m.inner.Register(ctx, in)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*models.User, *auth.TokenPair, error) {
	if m.loginErr != nil {
		return nil, nil, m.loginErr
	}
	return m.inner.Login(ctx, email, password)
}

func (m *mockAuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	return m.inner.GetUser(ctx, userID)
}

func (m *mockAuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return m.inner.ChangePassword(ctx, userID, currentPassword, newPassword)
}

func (m *mockAuthService) ChangeEmail(ctx context.Context, userID, currentPassword, newEmail string) (*models.User, error) {
	return m.inner.ChangeEmail(ctx, userID, currentPassword, newEmail)
}

func (m *mockAuthService) Tokens() *auth.TokenService {
	return m.inner.Tokens()
}

func newAuthHandler(t *testing.T) *AuthHandler {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	svc := auth.NewService(db, auth.NewTokenService("secret", time.Minute, time.Hour))
	return &AuthHandler{Service: svc}
}

func TestAuthHandlerRegisterErrors(t *testing.T) {
	h := newAuthHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Register(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}

	conflictHandler := newAuthHandler(t)
	_, r := gin.CreateTestContext(httptest.NewRecorder())
	r.POST("/register", conflictHandler.Register)
	body := `{"email":"dup@test.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first register: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req2)
	if w.Code != http.StatusConflict {
		t.Fatalf("conflict: %d %s", w.Code, w.Body.String())
	}

	db := testutil.OpenSQLiteDB(t)
	inner := auth.NewService(db, auth.NewTokenService("secret", time.Minute, time.Hour))
	mockH := &AuthHandler{Service: &mockAuthService{regErr: auth.ErrValidation, inner: inner}}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@b.com","password":"password123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	mockH.Register(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("service validation: %d", w.Code)
	}
}

func TestAuthHandlerRegisterGenericError(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	inner := auth.NewService(db, auth.NewTokenService("secret", time.Minute, time.Hour))
	h := &AuthHandler{Service: &mockAuthService{regErr: errors.New("boom"), inner: inner}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@b.com","password":"password123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Register(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestAuthHandlerRegisterInternal(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := auth.NewService(db, auth.NewTokenService("secret", time.Minute, time.Hour))
	testutil.CloseDB(t, db)
	h := &AuthHandler{Service: svc}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"x@y.com","password":"password123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Register(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestAuthHandlerLoginGenericError(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	inner := auth.NewService(db, auth.NewTokenService("secret", time.Minute, time.Hour))
	h := &AuthHandler{Service: &mockAuthService{loginErr: errors.New("boom"), inner: inner}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"a@b.com","password":"password123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestAuthHandlerLoginInternal(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := auth.NewService(db, auth.NewTokenService("secret", time.Minute, time.Hour))
	testutil.CloseDB(t, db)
	h := &AuthHandler{Service: svc}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"x@y.com","password":"password123"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestAuthHandlerLoginSuccess(t *testing.T) {
	h := newAuthHandler(t)
	ctx := context.Background()
	_, _, err := h.Service.Register(ctx, auth.RegisterInput{
		Email:    "login-ok@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(
		`{"email":"login-ok@test.com","password":"password123"}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	if w.Code != http.StatusOK {
		t.Fatalf("login success: %d %s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerLoginErrors(t *testing.T) {
	h := newAuthHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"not-an-email"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"email":"x@y.com","password":"nope"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Login(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized: %d", w.Code)
	}
}

func TestAuthHandlerRefreshErrors(t *testing.T) {
	h := newAuthHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Refresh(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"refresh_token":"bad"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Refresh(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized: %d", w.Code)
	}
}

func TestDefaultGuestTokenIssue(t *testing.T) {
	h := newAuthHandler(t)
	token, expiresIn, err := defaultGuestTokenIssue(h.Service, "device-1")
	if err != nil || token == "" || expiresIn <= 0 {
		t.Fatalf("guest issue: err=%v token=%q expires=%d", err, token, expiresIn)
	}

	auth.BreakSigningForTest(t)

	if _, _, err := defaultGuestTokenIssue(h.Service, "device-2"); err == nil {
		t.Fatal("expected guest token issue error")
	}
}

func TestAuthHandlerGuestErrors(t *testing.T) {
	h := newAuthHandler(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Guest(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}

	old := guestTokenIssue
	t.Cleanup(func() { guestTokenIssue = old })
	guestTokenIssue = func(_ authService, _ string) (string, int64, error) {
		return "", 0, errors.New("fail")
	}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"device_id":"d1"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Guest(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestAuthHandlerLogout(t *testing.T) {
	h := newAuthHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.Logout(c)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", w.Code)
	}
}

func TestTokenFromPair(t *testing.T) {
	resp := tokenFromPair(&auth.TokenPair{AccessToken: "a", RefreshToken: "r", ExpiresIn: 10})
	if resp.TokenType != "Bearer" || resp.AccessToken != "a" {
		t.Fatalf("resp: %+v", resp)
	}
}
