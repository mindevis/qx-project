package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

type testManifestProvider struct{}

func (testManifestProvider) BuildInstanceManifest(_ context.Context, instanceID, name, mcVersion, loader, loaderVersion string) (*mcmanifest.InstanceLaunchManifest, error) {
	return &mcmanifest.InstanceLaunchManifest{
		InstanceID: instanceID,
		Name:       name,
		MCVersion:  mcVersion,
		Loader:     loader,
		MainClass:  "net.minecraft.client.main.Main",
	}, nil
}

type failingManifestProvider struct{ err error }

func (p failingManifestProvider) BuildInstanceManifest(_ context.Context, instanceID, name, mcVersion, loader, loaderVersion string) (*mcmanifest.InstanceLaunchManifest, error) {
	return nil, p.err
}

func newLaunchHandler(t *testing.T) (*LaunchRequestsHandler, *launcher.Service, *auth.TokenService) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	svc.SetManifestProvider(testManifestProvider{})
	return &LaunchRequestsHandler{Service: svc, Tokens: tokens}, svc, tokens
}

func TestLaunchRequestsHandlerCreateWithLinkedDevice(t *testing.T) {
	h, svc, tokens := newLaunchHandler(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-web"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "dev-web", UserID: "user-web"}); err != nil {
		t.Fatalf("link: %v", err)
	}
	pair, _ := tokens.IssueUserTokens("user-web", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)
	owner := launcher.Owner{UserID: claims.UserID}

	inst, err := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("instance: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"instance_id": inst.ID})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, claims.UserID)
	h.Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create launch: %d %s", w.Code, w.Body.String())
	}
}

func TestLaunchRequestsHandlerCreateNoDevice(t *testing.T) {
	h, svc, tokens := newLaunchHandler(t)
	ctx := context.Background()
	owner := launcher.Owner{UserID: "user-1"}
	inst, err := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("instance: %v", err)
	}
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)

	body, _ := json.Marshal(map[string]string{"instance_id": inst.ID})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, claims.UserID)
	h.Create(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("forbidden: %d %s", w.Code, w.Body.String())
	}
}

func TestLaunchRequestsHandlerPending(t *testing.T) {
	h, svc, _ := newLaunchHandler(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-poll"})
	_, _ = svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "dev-poll", UserID: "user-poll"})
	owner := launcher.Owner{UserID: "user-poll"}
	inst, _ := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	_, _ = svc.CreateLaunchRequest(ctx, owner, launcher.CreateLaunchRequestInput{
		InstanceID: inst.ID, DeviceID: "dev-poll",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(DeviceIDKey, "dev-poll")
	h.Pending(c)
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}
}

func TestLaunchRequestsHandlerGet(t *testing.T) {
	h, svc, tokens := newLaunchHandler(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-get"})
	_, _ = svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "dev-get", UserID: "user-get"})
	pair, _ := tokens.IssueUserTokens("user-get", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)
	owner := launcher.Owner{UserID: claims.UserID}
	inst, _ := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, _ := svc.CreateLaunchRequest(ctx, owner, launcher.CreateLaunchRequestInput{
		InstanceID: inst.ID, DeviceID: "dev-get",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, claims.UserID)
	h.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
}

func TestLaunchRequestsHandlerUpdate(t *testing.T) {
	h, svc, _ := newLaunchHandler(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-patch"})
	_, _ = svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "dev-patch", UserID: "user-patch"})
	owner := launcher.Owner{UserID: "user-patch"}
	inst, _ := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, _ := svc.CreateLaunchRequest(ctx, owner, launcher.CreateLaunchRequestInput{
		InstanceID: inst.ID, DeviceID: "dev-patch",
	})

	body, _ := json.Marshal(map[string]any{"status": "running", "pid": 1234})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(DeviceIDKey, "dev-patch")
	h.Update(c)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}
}

func TestLaunchRequestsHandlerGetManifestFailure(t *testing.T) {
	h, svc, tokens := newLaunchHandler(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-manifest-h"})
	_, _ = svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "dev-manifest-h", UserID: "user-manifest-h"})
	pair, _ := tokens.IssueUserTokens("user-manifest-h", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)
	owner := launcher.Owner{UserID: claims.UserID}
	inst, _ := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, _ := svc.CreateLaunchRequest(ctx, owner, launcher.CreateLaunchRequestInput{
		InstanceID: inst.ID, DeviceID: "dev-manifest-h",
	})

	svc.SetManifestProvider(failingManifestProvider{err: fmt.Errorf("upstream unavailable")})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(UserIDKey, claims.UserID)
	h.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get manifest failure: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Status    string  `json:"status"`
		ErrorCode *string `json:"error_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body.Status != models.LaunchStatusFailed || body.ErrorCode == nil || *body.ErrorCode != "MANIFEST_UNAVAILABLE" {
		t.Fatalf("response: %+v", body)
	}
}

func TestLaunchRequestsHandlerCreateWithXDeviceToken(t *testing.T) {
	h, svc, tokens := newLaunchHandler(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-header"})
	_, _ = svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "dev-header", UserID: "user-header"})
	pair, _ := tokens.IssueUserTokens("user-header", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)
	owner := launcher.Owner{UserID: claims.UserID}
	inst, _ := svc.CreateInstance(ctx, owner, launcher.CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	status, _ := svc.DeviceStatus(ctx, "dev-header")
	deviceToken := *status.DeviceToken

	body, _ := json.Marshal(map[string]string{"instance_id": inst.ID})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Device-Token", deviceToken)
	c.Set(UserIDKey, claims.UserID)
	h.Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create with device header: %d %s", w.Code, w.Body.String())
	}
}
