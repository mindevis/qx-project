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
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func newReleaseHandler(t *testing.T) *ReleaseHandler {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	svc.SetRelease("1.2.3", "http://localhost:5173/downloads/qx-launcher.exe")
	return &ReleaseHandler{Service: svc}
}

func TestReleaseHandlerGet(t *testing.T) {
	h := newReleaseHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("release: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp["version"] != "1.2.3" || resp["download_url"] == "" {
		t.Fatalf("unexpected release: %#v", resp)
	}
}

func newUpdateHandler(t *testing.T) (*UpdateRequestsHandler, *launcher.Service) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	svc.SetRelease("2.0.0", "/downloads/qx-launcher.exe")
	return &UpdateRequestsHandler{Service: svc}, svc
}

func TestUpdateRequestsHandlerFlow(t *testing.T) {
	h, svc := newUpdateHandler(t)
	ctx := context.Background()

	ver := "0.9.0"
	_, err := svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{
		DeviceID:        "upd-dev",
		LauncherVersion: ver,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "upd-dev", UserID: "user-upd"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(UserIDKey, "user-upd")
	h.Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(DeviceIDKey, "upd-dev")
	h.Pending(c)
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}
	var pending struct {
		Item *launcher.UpdateRequestView `json:"item"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pending); err != nil || pending.Item == nil {
		t.Fatalf("pending body: %v item=%v", err, pending.Item)
	}
	if pending.Item.Version != "2.0.0" {
		t.Fatalf("version: %s", pending.Item.Version)
	}

	body, _ := json.Marshal(map[string]string{
		"status":           "completed",
		"launcher_version": "2.0.0",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "upd-dev"}}
	c.Set(DeviceIDKey, "upd-dev")
	h.Complete(c)
	if w.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", w.Code, w.Body.String())
	}

	info, err := svc.UserLinkedDevice(ctx, "user-upd")
	if err != nil {
		t.Fatalf("linked device: %v", err)
	}
	if info.LauncherVersion == nil || *info.LauncherVersion != "2.0.0" {
		t.Fatalf("version not updated: %#v", info.LauncherVersion)
	}
}

func TestUpdateRequestsHandlerCreateNoDevice(t *testing.T) {
	h, _ := newUpdateHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(UserIDKey, "missing-user")
	h.Create(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
