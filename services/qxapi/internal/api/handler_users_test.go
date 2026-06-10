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
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestUsersHandlerMe(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := auth.NewService(db, tokens)
	h := &UsersHandler{Service: svc}

	user, _, err := svc.Register(context.Background(), auth.RegisterInput{
		Email:    "me@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, user.ID)
	h.Me(c)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, "00000000-0000-0000-0000-000000000099")
	h.Me(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}

	testutil.CloseDB(t, db)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, user.ID)
	h.Me(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestUsersHandlerChangePasswordAndEmail(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := auth.NewService(db, tokens)
	h := &UsersHandler{Service: svc}

	user, _, err := svc.Register(context.Background(), auth.RegisterInput{
		Email:    "change@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "password123",
		"new_password":     "newpassword456",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangePassword(c)
	if w.Code != http.StatusNoContent {
		t.Fatalf("change password: %d %s", w.Code, w.Body.String())
	}

	body, _ = json.Marshal(map[string]string{
		"current_password": "newpassword456",
		"email":            "updated@test.com",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangeEmail(c)
	if w.Code != http.StatusOK {
		t.Fatalf("change email: %d %s", w.Code, w.Body.String())
	}
}

func TestUsersHandlerChangePasswordErrors(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := auth.NewService(db, tokens)
	h := &UsersHandler{Service: svc}

	user, _, err := svc.Register(context.Background(), auth.RegisterInput{
		Email:    "pw-err@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangePassword(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "wrong",
		"new_password":     "newpassword456",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangePassword(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", w.Code)
	}

	body, _ = json.Marshal(map[string]string{
		"current_password": "password123",
		"new_password":     "newpassword456",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "00000000-0000-0000-0000-000000000099")
	h.ChangePassword(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}

	testutil.CloseDB(t, db)
	body, _ = json.Marshal(map[string]string{
		"current_password": "password123",
		"new_password":     "newpassword456",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangePassword(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

func TestUsersHandlerChangeEmailErrors(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := auth.NewService(db, tokens)
	h := &UsersHandler{Service: svc}

	user, _, err := svc.Register(context.Background(), auth.RegisterInput{
		Email:    "email-err@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangeEmail(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}

	body, _ := json.Marshal(map[string]string{
		"current_password": "wrong",
		"email":            "new@test.com",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangeEmail(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", w.Code)
	}

	_, _, err = svc.Register(context.Background(), auth.RegisterInput{
		Email:    "taken2@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register taken: %v", err)
	}
	body, _ = json.Marshal(map[string]string{
		"current_password": "password123",
		"email":            "taken2@test.com",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangeEmail(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("email taken: %d", w.Code)
	}

	body, _ = json.Marshal(map[string]string{
		"current_password": "password123",
		"email":            "updated2@test.com",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "00000000-0000-0000-0000-000000000099")
	h.ChangeEmail(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}

	testutil.CloseDB(t, db)
	body, _ = json.Marshal(map[string]string{
		"current_password": "password123",
		"email":            "updated2@test.com",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, user.ID)
	h.ChangeEmail(c)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("internal: %d", w.Code)
	}
}

type stubUsersAuthService struct {
	changePassword func(context.Context, string, string, string) error
	changeEmail    func(context.Context, string, string, string) (*models.User, error)
}

func (s stubUsersAuthService) Register(context.Context, auth.RegisterInput) (*models.User, *auth.TokenPair, error) {
	panic("not implemented")
}
func (s stubUsersAuthService) Login(context.Context, string, string) (*models.User, *auth.TokenPair, error) {
	panic("not implemented")
}
func (s stubUsersAuthService) GetUser(context.Context, string) (*models.User, error) {
	panic("not implemented")
}
func (s stubUsersAuthService) ChangePassword(ctx context.Context, userID, current, next string) error {
	return s.changePassword(ctx, userID, current, next)
}
func (s stubUsersAuthService) ChangeEmail(ctx context.Context, userID, current, email string) (*models.User, error) {
	return s.changeEmail(ctx, userID, current, email)
}
func (stubUsersAuthService) Tokens() *auth.TokenService { return nil }

func TestUsersHandlerChangePasswordServiceValidation(t *testing.T) {
	h := &UsersHandler{Service: stubUsersAuthService{
		changePassword: func(context.Context, string, string, string) error {
			return auth.ErrValidation
		},
	}}

	body, _ := json.Marshal(map[string]string{
		"current_password": "password123",
		"new_password":     "newpassword456",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "user-id")
	h.ChangePassword(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}
}

func TestUsersHandlerChangeEmailServiceValidation(t *testing.T) {
	h := &UsersHandler{Service: stubUsersAuthService{
		changeEmail: func(context.Context, string, string, string) (*models.User, error) {
			return nil, auth.ErrValidation
		},
	}}

	body, _ := json.Marshal(map[string]string{
		"current_password": "password123",
		"email":            "new@test.com",
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "user-id")
	h.ChangeEmail(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}
}
