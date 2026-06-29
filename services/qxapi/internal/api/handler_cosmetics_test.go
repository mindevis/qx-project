package api

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func newCosmeticsHandler(t *testing.T) (*CosmeticsHandler, *auth.TokenService) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	if err := db.AutoMigrate(&models.UserCosmetics{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dir := t.TempDir()
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := cosmetics.NewService(db, cosmetics.Config{DataDir: dir, PublicAPIURL: "http://localhost:3000"})
	return &CosmeticsHandler{Service: svc}, tokens
}

func TestCosmeticsHandlerGetAndEquip(t *testing.T) {
	h, tokens := newCosmeticsHandler(t)
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(UserIDKey, claims.UserID)
	h.GetMine(c)
	if w.Code != http.StatusOK {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}

	body, _ := json.Marshal(map[string]string{
		"skin_model": "alex",
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(UserIDKey, claims.UserID)
	h.Equip(c)
	if w.Code != http.StatusOK {
		t.Fatalf("equip: %d %s", w.Code, w.Body.String())
	}
}

func TestCosmeticsHandlerUploadSkin(t *testing.T) {
	h, tokens := newCosmeticsHandler(t)
	pair, _ := tokens.IssueUserTokens("user-1", "u@test.com")
	claims, _ := tokens.Parse(pair.AccessToken)

	png := testSkinPNG(64, 64)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("skin", "skin.png")
	_, _ = part.Write(png)
	_ = writer.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", &buf)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set(UserIDKey, claims.UserID)
	h.UploadSkin(c)
	if w.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Params = gin.Params{{Key: "userId", Value: "user-1.png"}}
	h.ServeSkin(c)
	if w.Code != http.StatusOK {
		t.Fatalf("serve: %d", w.Code)
	}
}

func testSkinPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
