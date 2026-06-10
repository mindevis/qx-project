package auth

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

type brokenIssuer struct{}

func (brokenIssuer) IssueUserTokens(string, string) (*TokenPair, error) {
	return nil, errors.New("token issue failed")
}

func newTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, NewTokenService("test-secret", 0, 0))
	return svc, db
}

func TestServiceRegisterLoginGetUser(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	name := "tester"

	user, pair, err := svc.Register(ctx, RegisterInput{
		Email:    "User@Example.COM ",
		Password: "password123",
		Username: &name,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "user@example.com" || pair.AccessToken == "" {
		t.Fatalf("user=%+v", user)
	}

	_, pair2, err := svc.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if pair2.RefreshToken == "" {
		t.Fatal("expected tokens on login")
	}

	got, err := svc.GetUser(ctx, user.ID)
	if err != nil || got.Email != user.Email {
		t.Fatalf("get user: %v %+v", err, got)
	}

	if svc.Tokens() == nil {
		t.Fatal("expected token service")
	}
}

func TestServiceRegisterWithoutUsername(t *testing.T) {
	svc, _ := newTestService(t)
	user, pair, err := svc.Register(context.Background(), RegisterInput{
		Email:    "no-name@test.com",
		Password: "password123",
	})
	if err != nil || user.Username != nil || pair.AccessToken == "" {
		t.Fatalf("register: err=%v user=%+v", err, user)
	}
}

func TestServiceRegisterValidation(t *testing.T) {
	svc, _ := newTestService(t)
	_, _, err := svc.Register(context.Background(), RegisterInput{Email: "", Password: "short"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func TestServiceRegisterEmailTaken(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, _, err := svc.Register(ctx, RegisterInput{Email: "dup@test.com", Password: "password123"})
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, _, err = svc.Register(ctx, RegisterInput{Email: "dup@test.com", Password: "password456"})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected taken, got %v", err)
	}
}

func TestServiceRegisterDBErrorOnLookup(t *testing.T) {
	svc, db := newTestService(t)
	testutil.CloseDB(t, db)
	_, _, err := svc.Register(context.Background(), RegisterInput{
		Email:    "x@test.com",
		Password: "password123",
	})
	if err == nil || errors.Is(err, ErrValidation) || errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceRegisterTokenIssueError(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, brokenIssuer{})
	_, _, err := svc.Register(context.Background(), RegisterInput{
		Email:    "token@test.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected token issue error")
	}
}

func TestServiceLoginErrors(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, _, err := svc.Login(ctx, "missing@test.com", "password123")
	if !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("expected invalid login, got %v", err)
	}

	_, _, err = svc.Register(ctx, RegisterInput{Email: "ok@test.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_, _, err = svc.Login(ctx, "ok@test.com", "wrongpass")
	if !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("expected wrong password, got %v", err)
	}
}

func TestServiceLoginDBError(t *testing.T) {
	svc, db := newTestService(t)
	testutil.CloseDB(t, db)
	_, _, err := svc.Login(context.Background(), "a@b.com", "password123")
	if err == nil || errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestServiceLoginTokenIssueError(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, _, err := svc.Register(ctx, RegisterInput{Email: "login-token@test.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	svc.tokens = brokenIssuer{}
	_, _, err = svc.Login(ctx, "login-token@test.com", "password123")
	if err == nil {
		t.Fatal("expected token issue error on login")
	}
}

func TestServiceTokensNonJWT(t *testing.T) {
	svc := NewService(nil, brokenIssuer{})
	if svc.Tokens() != nil {
		t.Fatal("expected nil tokens for non-jwt issuer")
	}
}

func TestServiceRegisterHashError(t *testing.T) {
	old := hashPasswordFn
	hashPasswordFn = func([]byte, int) ([]byte, error) {
		return nil, errors.New("hash failed")
	}
	t.Cleanup(func() { hashPasswordFn = old })

	svc, _ := newTestService(t)
	_, _, err := svc.Register(context.Background(), RegisterInput{
		Email:    "hash-fail@test.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected hash error")
	}
}

func TestServiceRegisterCreateError(t *testing.T) {
	old := createUser
	createUser = func(*gorm.DB, *models.User) error {
		return errors.New("create failed")
	}
	t.Cleanup(func() { createUser = old })

	svc, _ := newTestService(t)
	_, _, err := svc.Register(context.Background(), RegisterInput{
		Email:    "create-fail@test.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestServiceChangePasswordAndEmail(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "profile@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, _, err := svc.Login(ctx, "profile@test.com", "newpassword456"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}

	updated, err := svc.ChangeEmail(ctx, user.ID, "newpassword456", "  NEW-EMAIL@test.com  ")
	if err != nil || updated.Email != "new-email@test.com" {
		t.Fatalf("change email: err=%v user=%+v", err, updated)
	}

	if err := svc.ChangePassword(ctx, user.ID, "wrongpass", "anotherpass1"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected wrong password, got %v", err)
	}

	_, err = svc.ChangeEmail(ctx, user.ID, "newpassword456", "")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}

	_, err = svc.ChangeEmail(ctx, user.ID, "wrongpass", "another@test.com")
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected wrong password on email change, got %v", err)
	}

	_, _, err = svc.Register(ctx, RegisterInput{Email: "taken@test.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register taken: %v", err)
	}
	_, err = svc.ChangeEmail(ctx, user.ID, "newpassword456", "taken@test.com")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected email taken, got %v", err)
	}
}

func TestServiceGetUserNotFound(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.GetUser(context.Background(), "00000000-0000-0000-0000-000000000099")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestServiceChangePasswordErrors(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "pw-svc@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.ChangePassword(ctx, user.ID, "password123", "short"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}

	if err := svc.ChangePassword(ctx, "00000000-0000-0000-0000-000000000099", "password123", "newpassword456"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	old := hashPasswordFn
	hashPasswordFn = func([]byte, int) ([]byte, error) {
		return nil, errors.New("hash failed")
	}
	t.Cleanup(func() { hashPasswordFn = old })

	if err := svc.ChangePassword(ctx, user.ID, "password123", "newpassword456"); err == nil {
		t.Fatal("expected hash error")
	}
}

func TestServiceChangeEmailEdgeCases(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "same-email@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	same, err := svc.ChangeEmail(ctx, user.ID, "password123", "  SAME-EMAIL@test.com  ")
	if err != nil || same.Email != user.Email {
		t.Fatalf("same email: err=%v user=%+v", err, same)
	}

	err = checkEmailAvailable(svc.db, ctx, "other@test.com")
	if err != nil {
		t.Fatalf("available email: %v", err)
	}
	_, _, err = svc.Register(ctx, RegisterInput{Email: "taken2@test.com", Password: "password123"})
	if err != nil {
		t.Fatalf("register taken: %v", err)
	}
	if err := checkEmailAvailable(svc.db, ctx, "taken2@test.com"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected taken, got %v", err)
	}

	if _, err := svc.ChangeEmail(ctx, "00000000-0000-0000-0000-000000000099", "password123", "x@test.com"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	testutil.CloseDB(t, db)
	_, err = svc.ChangeEmail(ctx, user.ID, "password123", "closed-db@test.com")
	if err == nil {
		t.Fatal("expected db error")
	}
}

func TestCheckEmailAvailableDBError(t *testing.T) {
	svc, db := newTestService(t)
	testutil.CloseDB(t, db)

	err := checkEmailAvailable(db, context.Background(), "x@y.com")
	if err == nil || errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected db error, got %v", err)
	}
	_ = svc
}

func TestServiceChangeEmailLookupError(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "lookup@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	oldLookup := checkEmailAvailable
	checkEmailAvailable = func(*gorm.DB, context.Context, string) error {
		return errors.New("lookup failed")
	}
	t.Cleanup(func() { checkEmailAvailable = oldLookup })

	_, err = svc.ChangeEmail(ctx, user.ID, "password123", "other@test.com")
	if err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestServiceChangeEmailValidationWhitespace(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "ws@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err = svc.ChangeEmail(ctx, user.ID, "password123", "   ")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func TestServiceChangeEmailReloadError(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "reload@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	oldReload := reloadUserAfterEmailChange
	reloadUserAfterEmailChange = func(*Service, context.Context, string) (*models.User, error) {
		return nil, errors.New("reload failed")
	}
	t.Cleanup(func() { reloadUserAfterEmailChange = oldReload })

	_, err = svc.ChangeEmail(ctx, user.ID, "password123", "reloaded@test.com")
	if err == nil || err.Error() != "reload failed" {
		t.Fatalf("expected reload error, got %v", err)
	}
}

func TestServiceChangeEmailUpdateError(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, RegisterInput{
		Email:    "update@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	oldUpdate := updateUserEmail
	updateUserEmail = func(*gorm.DB, context.Context, string, string) error {
		return errors.New("update failed")
	}
	t.Cleanup(func() { updateUserEmail = oldUpdate })

	_, err = svc.ChangeEmail(ctx, user.ID, "password123", "other@test.com")
	if err == nil || err.Error() != "update failed" {
		t.Fatalf("expected update error, got %v", err)
	}
}
