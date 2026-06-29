package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func newSessionHandler(t *testing.T) (*SessionHandler, *cosmetics.Service, string) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	if err := db.AutoMigrate(&models.UserCosmetics{}, &models.MojangLink{}, &models.OfflineProfile{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dir := t.TempDir()
	svc := cosmetics.NewService(db, cosmetics.Config{
		DataDir:             dir,
		PublicAPIURL:        "http://localhost:3000",
		SkinServerPublicURL: "http://localhost:3000",
	})
	gameUUID := uuid.NewString()
	userID := "user-1"
	if err := db.Create(&models.MojangLink{
		UserID:          userID,
		MinecraftUUID:   gameUUID,
		Username:        "TestPlayer",
		RefreshTokenEnc: []byte("test"),
		LinkedAt:        time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("link: %v", err)
	}
	png := testSkinPNG(64, 64)
	if _, err := svc.UploadSkin(t.Context(), userID, png); err != nil {
		t.Fatalf("upload: %v", err)
	}
	return &SessionHandler{Service: svc}, svc, gameUUID
}

func TestSessionHandlerProfile(t *testing.T) {
	h, _, gameUUID := newSessionHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "uuid", Value: gameUUID}}
	h.Profile(c)
	if w.Code != http.StatusOK {
		t.Fatalf("profile: %d %s", w.Code, w.Body.String())
	}
	var profile map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &profile); err != nil {
		t.Fatalf("json: %v", err)
	}
	if profile["name"] != "TestPlayer" {
		t.Fatalf("name: %v", profile["name"])
	}
	props, ok := profile["properties"].([]any)
	if !ok || len(props) == 0 {
		t.Fatalf("properties missing: %+v", profile)
	}
}

func TestSessionHandlerProfileNotFound(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	if err := db.AutoMigrate(&models.UserCosmetics{}, &models.MojangLink{}, &models.OfflineProfile{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := cosmetics.NewService(db, cosmetics.Config{
		DataDir:             t.TempDir(),
		PublicAPIURL:        "http://localhost:3000",
		SkinServerPublicURL: "http://localhost:3000",
	})
	h := &SessionHandler{Service: svc}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "uuid", Value: uuid.NewString()}}
	h.Profile(c)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d %s", w.Code, w.Body.String())
	}
}

func TestSessionHandlerMeta(t *testing.T) {
	h, _, _ := newSessionHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	h.Meta(c)
	if w.Code != http.StatusOK {
		t.Fatalf("meta: %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("QX Skin Server")) {
		t.Fatalf("meta body: %s", w.Body.String())
	}
}

func strPtr(s string) *string { return &s }
