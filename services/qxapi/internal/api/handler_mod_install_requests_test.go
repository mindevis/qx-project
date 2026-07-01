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

func newModInstallHandler(t *testing.T) (*ModInstallRequestsHandler, *launcher.Service) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := launcher.NewService(db, tokens, "http://localhost:5173")
	return &ModInstallRequestsHandler{Service: svc, Tokens: tokens}, svc
}

func TestModInstallRequestsHandlerFlow(t *testing.T) {
	h, svc := newModInstallHandler(t)
	ctx := context.Background()

	if _, err := svc.RegisterDevice(ctx, launcher.RegisterDeviceInput{DeviceID: "mod-dev"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, launcher.LinkDeviceInput{DeviceID: "mod-dev", UserID: "user-mod"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	inst, err := svc.CreateInstance(ctx, launcher.Owner{UserID: "user-mod"}, launcher.CreateInstanceInput{
		Name:          "Forge",
		MCVersion:     "1.21",
		Loader:        "forge",
		LoaderVersion: "47.0.0",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	body := `{
		"instance_id":"` + inst.ID + `",
		"source":"modrinth",
		"project_id":"sodium",
		"project_name":"Sodium",
		"version_id":"ver-1",
		"version_number":"0.5.0",
		"filename":"sodium.jar",
		"download_url":"https://example/sodium.jar",
		"resource_type":"mod"
	}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, "user-mod")
	h.Create(c)
	if w.Code != http.StatusCreated {
		t.Fatalf("create mod install: %d %s", w.Code, w.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(DeviceIDKey, "mod-dev")
	h.Pending(c)
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}

	patch := `{"status":"completed"}`
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(patch))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: created.ID}}
	c.Set(DeviceIDKey, "mod-dev")
	h.Update(c)
	if w.Code != http.StatusOK {
		t.Fatalf("update: %d %s", w.Code, w.Body.String())
	}

	items, err := svc.ListInstanceResources(ctx, launcher.Owner{UserID: "user-mod"}, inst.ID)
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(items) != 1 || items[0].ProjectName != "Sodium" {
		t.Fatalf("expected installed sodium, got %+v", items)
	}
}
