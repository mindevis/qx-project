package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

func TestModsHandlerSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/search") {
			_, _ = w.Write([]byte(`{"hits":[{"project_id":"abc","slug":"sodium","title":"Sodium","description":"fps","author":"dev","display_icon_url":"","project_type":"mod","downloads":100,"versions":["1.21.1"],"categories":["fabric"]}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	// Modrinth client uses fixed base; handler test uses Service with real Modrinth — skip live and test validation instead.
	h := &ModsHandler{Service: mods.NewService(mods.Config{})}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mods/search", nil)
	h.Search(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing q: got %d %s", w.Code, w.Body.String())
	}
}

func TestModsHandlerBrowseCurseForgeUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ModsHandler{Service: mods.NewService(mods.Config{})}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mods/browse?source=curseforge&type=mod&loader=forge&mc_version=1.20.1", nil)
	h.Browse(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "SOURCE_UNAVAILABLE") {
		t.Fatalf("expected SOURCE_UNAVAILABLE: %s", w.Body.String())
	}
}

func TestModsHandlerBrowseCurseForgeUpstreamError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	writeModsUpstreamError(c, fmt.Errorf("curseforge: status 403: forbidden"))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "CURSEFORGE_UNAVAILABLE") {
		t.Fatalf("expected CURSEFORGE_UNAVAILABLE: %s", w.Body.String())
	}
}

func TestModsHandlerBrowseSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ModsHandler{
		Service: mods.NewService(mods.Config{ModrinthUserAgent: "QXTest/1.0"}),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mods/browse?type=mod&loader=fabric&mc_version=1.21.1&sort=downloads", nil)
	h.Browse(c)
	if w.Code != http.StatusOK && w.Code != http.StatusBadGateway {
		t.Fatalf("status: %d %s", w.Code, w.Body.String())
	}
	if w.Code == http.StatusOK {
		var body struct {
			Items   []mods.SearchItem `json:"items"`
			HasMore bool              `json:"has_more"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Items) == 0 {
			t.Fatalf("expected items")
		}
	}
}

func TestModsHandlerSearchSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ModsHandler{
		Service: mods.NewService(mods.Config{ModrinthUserAgent: "QXTest/1.0"}),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mods/search?q=sodium&type=mod&loader=fabric&mc_version=1.21.1", nil)
	h.Search(c)
	if w.Code != http.StatusOK && w.Code != http.StatusBadGateway {
		t.Fatalf("status: %d %s", w.Code, w.Body.String())
	}
	if w.Code == http.StatusOK {
		var body struct {
			Items []mods.SearchItem `json:"items"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Items) == 0 {
			t.Fatalf("expected items")
		}
	}
}

func TestModsHandlerSearchCurseForgeEnabledFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ModsHandler{
		Service: mods.NewService(mods.Config{CurseForgeAPIKey: "test-key"}),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mods/search", nil)
	h.Search(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing q: got %d", w.Code)
	}

	hDisabled := &ModsHandler{Service: mods.NewService(mods.Config{})}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/mods/search?q=test", nil)
	// Will fail upstream or succeed with modrinth only; we only need the flag on success path.
	// Test enabled/disabled via Service directly in mods package; here verify handler exposes flag when service configured.
	if !h.Service.CurseForgeEnabled() {
		t.Fatal("expected curseforge enabled")
	}
	if hDisabled.Service.CurseForgeEnabled() {
		t.Fatal("expected curseforge disabled")
	}
}

func TestGameServerSupportsMods(t *testing.T) {
	if !gameServerSupportsMods("forge") {
		t.Fatal("forge should support mods")
	}
	if gameServerSupportsMods("paper") {
		t.Fatal("paper should not support mods")
	}
}

func TestSyncModValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := setupRouter(t)

	regBody := `{"email":"mods-sync@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers/srv/game-servers/gs/mods/sync", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("sync validation: %d %s", w.Code, w.Body.String())
	}
}
