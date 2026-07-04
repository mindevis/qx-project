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

func newPrepareHandler(t *testing.T) (*PrepareRequestsHandler, *launcher.Service) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	return &PrepareRequestsHandler{Service: svc}, svc
}

func TestPrepareRequestsHandlerFlow(t *testing.T) {
	h, svc := newPrepareHandler(t)
	ctx := context.Background()

	if _, err := svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "prep-dev"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "prep-dev", UserID: "user-prep"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	createRes, err := svc.CreateInstance(ctx, launcher.Owner{UserID: "user-prep"}, launcher.CreateInstanceInput{
		Name:          "Forge",
		MCVersion:     "1.21",
		Loader:        models.LoaderForge,
		LoaderVersion: "47.0.0",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if createRes.PrepareRequestID == nil || *createRes.PrepareRequestID == "" {
		t.Fatal("expected prepare request id on create")
	}
	prepareID := *createRes.PrepareRequestID

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(DeviceIDKey, "prep-dev")
	h.Pending(c)
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}

	var pending struct {
		Item *struct {
			ID         string `json:"id"`
			InstanceID string `json:"instance_id"`
			Status     string `json:"status"`
		} `json:"item"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pending); err != nil {
		t.Fatalf("decode pending: %v", err)
	}
	if pending.Item == nil || pending.Item.ID != prepareID {
		t.Fatalf("expected pending prepare %s, got %+v", prepareID, pending.Item)
	}
	if pending.Item.InstanceID != createRes.Instance.ID {
		t.Fatalf("instance id mismatch")
	}

	patch := `{"status":"completed"}`
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(patch))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: prepareID}}
	c.Set(DeviceIDKey, "prep-dev")
	h.Update(c)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, "user-prep")
	c.Params = gin.Params{{Key: "id", Value: prepareID}}
	h.Get(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	var got struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Status != models.PrepareStatusCompleted {
		t.Fatalf("expected completed, got %s", got.Status)
	}
}

func TestCreateInstanceWithoutLinkedDeviceSkipsPrepare(t *testing.T) {
	_, svc := newPrepareHandler(t)
	ctx := context.Background()

	createRes, err := svc.CreateInstance(ctx, launcher.Owner{UserID: "user-no-device"}, launcher.CreateInstanceInput{
		Name:      "Vanilla",
		MCVersion: "1.21",
		Loader:    models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if createRes.PrepareRequestID != nil {
		t.Fatalf("expected no prepare request without linked device, got %s", *createRes.PrepareRequestID)
	}
}
