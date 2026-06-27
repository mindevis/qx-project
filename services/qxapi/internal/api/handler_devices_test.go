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

func newDevicesHandler(t *testing.T) *DevicesHandler {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	return &DevicesHandler{Service: svc}
}

func TestDevicesHandlerRegisterLinkStatus(t *testing.T) {
	h := newDevicesHandler(t)

	body, _ := json.Marshal(map[string]string{"device_id": "dev-1", "os": "windows"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Register(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "dev-1"}}
	h.Status(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}

	linkBody, _ := json.Marshal(map[string]string{"device_id": "dev-1"})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(linkBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "user-dev-1")
	h.Link(c)
	if w.Code != http.StatusOK {
		t.Fatalf("link: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "dev-1"}}
	h.Status(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status linked: %d", w.Code)
	}
}

func TestDevicesHandlerRegisterValidation(t *testing.T) {
	h := newDevicesHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Register(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation: %d", w.Code)
	}
}

func TestDevicesHandlerStatusNotFound(t *testing.T) {
	h := newDevicesHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "missing"}}
	h.Status(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("not found: %d", w.Code)
	}
}

func TestDevicesHandlerLinkAsUser(t *testing.T) {
	h := newDevicesHandler(t)
	ctx := context.Background()
	_, err := h.Service.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "dev-user"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"device_id": "dev-user"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "user-123")
	h.Link(c)
	if w.Code != http.StatusOK {
		t.Fatalf("link user: %d %s", w.Code, w.Body.String())
	}
}

func TestDevicesHandlerMeInstances(t *testing.T) {
	h := newDevicesHandler(t)
	ctx := context.Background()

	_, err := h.Service.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "me-dev"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := h.Service.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "me-dev", UserID: "user-me"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	device, _ := h.Service.DeviceStatus(ctx, "me-dev")
	_ = device

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(DeviceIDKey, "me-dev")
	h.MeInstances(c)
	if w.Code != http.StatusOK {
		t.Fatalf("me instances: %d %s", w.Code, w.Body.String())
	}
}

func TestDevicesHandlerMe(t *testing.T) {
	h := newDevicesHandler(t)
	ctx := context.Background()

	_, err := h.Service.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "me-info"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := h.Service.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "me-info", UserID: "user-1"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(DeviceIDKey, "me-info")
	h.Me(c)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
}

func TestDevicesHandlerUserLinkedDevice(t *testing.T) {
	h := newDevicesHandler(t)
	ctx := context.Background()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, "user-no-device")
	h.UserLinkedDevice(c)
	if w.Code != http.StatusOK {
		t.Fatalf("unlinked: %d", w.Code)
	}

	_, _ = h.Service.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "user-dev"})
	_, _ = h.Service.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "user-dev", UserID: "user-linked"})

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, "user-linked")
	h.UserLinkedDevice(c)
	if w.Code != http.StatusOK {
		t.Fatalf("linked: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp["linked"] != true {
		t.Fatalf("body: %v err=%v", resp, err)
	}
}

func TestDevicesHandlerUnlinkSelf(t *testing.T) {
	h := newDevicesHandler(t)
	ctx := context.Background()

	_, err := h.Service.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "unlink-self"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := h.Service.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "unlink-self", UserID: "user-self"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(DeviceIDKey, "unlink-self")
	h.Unlink(c)
	if w.Code != http.StatusOK {
		t.Fatalf("unlink: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(DeviceIDKey, "unlink-self")
	h.Unlink(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("double unlink: %d", w.Code)
	}
}

func TestDevicesHandlerUnlinkOwner(t *testing.T) {
	h := newDevicesHandler(t)
	ctx := context.Background()

	_, err := h.Service.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "unlink-owner"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := h.Service.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "unlink-owner", UserID: "user-unlink"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(UserIDKey, "user-unlink")
	h.Unlink(c)
	if w.Code != http.StatusOK {
		t.Fatalf("unlink owner: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set(UserIDKey, "user-unlink")
	h.Unlink(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("no linked device: %d", w.Code)
	}
}

func TestDevicesHandlerUnlinkUnauthorized(t *testing.T) {
	h := newDevicesHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	h.Unlink(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized: %d", w.Code)
	}
}
