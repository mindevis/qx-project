package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

type captureDeployer struct {
	token    string
	serverID string
}

func (c *captureDeployer) Deploy(_ context.Context, serverID string, _ models.SSHCredential, agentToken string) error {
	c.token = agentToken
	c.serverID = serverID
	return nil
}

type noopDeployer struct{}

func (noopDeployer) Deploy(context.Context, string, models.SSHCredential, string) error {
	return nil
}

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return setupRouterWithDeploy(t, DeploySettings{DeployExecutor: noopDeployer{}})
}

func setupRouterWithDeploy(t *testing.T, deploy DeploySettings) *gin.Engine {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("test-secret", time.Minute, time.Hour)
	svc := auth.NewService(db, tokens)
	return NewRouter(db, svc, "http://localhost:5173", devSSHMasterKey(), deploy, MojangSettings{
		RedirectURI: "http://localhost:3000/api/v1/mojang/oauth/callback",
		WebBaseURL:  "http://localhost:5173",
		JWTSecret:   "test-secret",
	}, ModsSettings{}, CosmeticsSettings{DataDir: t.TempDir(), PublicAPIURL: "http://localhost:3000"}, LauncherSettings{
		Version:     "0.1.0-dev",
		DownloadURL: "http://localhost:5173/downloads/qx-launcher.exe",
	})
}

func TestDevSSHMasterKey(t *testing.T) {
	if len(devSSHMasterKey()) != 44 {
		t.Fatalf("unexpected dev key length")
	}
}

func TestRouterSwaggerUI(t *testing.T) {
	r := setupRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("swagger UI: got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "swagger") {
		t.Fatalf("expected swagger UI body")
	}
}

func TestRouterAuthFlow(t *testing.T) {
	r := setupRouter(t)

	regBody := `{"email":"router@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}

	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("json: %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	refresh := map[string]string{"refresh_token": tokens.RefreshToken}
	b, _ := json.Marshal(refresh)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", w.Code)
	}


	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("health: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ready: %d", w.Code)
	}
}

func TestRouterLauncherFlow(t *testing.T) {
	r := setupRouter(t)

	regBody := `{"email":"router-launcher@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var userTokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &userTokens); err != nil {
		t.Fatalf("json: %v", err)
	}

	devBody := `{"device_id":"router-dev","os":"windows"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/devices/register", bytes.NewBufferString(devBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("device register: %d %s", w.Code, w.Body.String())
	}

	linkBody := `{"device_id":"router-dev"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/devices/link", bytes.NewBufferString(linkBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("device link: %d %s", w.Code, w.Body.String())
	}

	instBody := `{"name":"Survival","mc_version":"1.21","loader":"vanilla"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewBufferString(instBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create instance: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list instances: %d %s", w.Code, w.Body.String())
	}

	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil || len(listResp.Items) == 0 {
		t.Fatalf("instances list: %v body=%s", err, w.Body.String())
	}
	instanceID := listResp.Items[0].ID

	profileBody := `{"username":"Steve"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/profiles", bytes.NewBufferString(profileBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create profile: %d %s", w.Code, w.Body.String())
	}
	var profileResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &profileResp); err != nil {
		t.Fatalf("profile json: %v", err)
	}

	launchBody := `{"instance_id":"` + instanceID + `","offline_profile_id":"` + profileResp.ID + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/launch-requests", bytes.NewBufferString(launchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create launch request: %d %s", w.Code, w.Body.String())
	}
	var launchResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &launchResp); err != nil {
		t.Fatalf("launch json: %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/launcher/devices/router-dev/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("device status: %d %s", w.Code, w.Body.String())
	}
	var statusResp struct {
		DeviceToken *string `json:"device_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil || statusResp.DeviceToken == nil {
		t.Fatalf("device token: err=%v body=%s", err, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/launcher/launch-requests/pending", nil)
	req.Header.Set("Authorization", "Bearer "+*statusResp.DeviceToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}

	patchBody := `{"status":"completed","exit_code":0}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/launcher/launch-requests/"+launchResp.ID, bytes.NewBufferString(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+*statusResp.DeviceToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update launch: %d %s", w.Code, w.Body.String())
	}
}


// Flow A (registered user): register → link device → instances → launch-bridge poll.
func TestRouterFlowA_RegisteredUserLauncher(t *testing.T) {
	r := setupRouter(t)

	regBody := `{"email":"flowa@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}

	var userTokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &userTokens); err != nil {
		t.Fatalf("json: %v", err)
	}

	devBody := `{"device_id":"flowa-dev","os":"windows"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/devices/register", bytes.NewBufferString(devBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("device register: %d %s", w.Code, w.Body.String())
	}

	linkBody := `{"device_id":"flowa-dev"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/devices/link", bytes.NewBufferString(linkBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("device link: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/me/launcher-device", nil)
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("launcher-device: %d %s", w.Code, w.Body.String())
	}
	var linked struct {
		Linked   bool   `json:"linked"`
		DeviceID string `json:"device_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &linked); err != nil || !linked.Linked || linked.DeviceID != "flowa-dev" {
		t.Fatalf("linked device: err=%v body=%s", err, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/launcher/devices/flowa-dev/status", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("device status: %d %s", w.Code, w.Body.String())
	}
	var statusResp struct {
		Status      string  `json:"status"`
		DeviceToken *string `json:"device_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil || statusResp.DeviceToken == nil {
		t.Fatalf("device token: err=%v body=%s", err, w.Body.String())
	}
	deviceToken := *statusResp.DeviceToken

	instBody := `{"name":"FlowA","mc_version":"1.21","loader":"vanilla"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/instances", bytes.NewBufferString(instBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create instance: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list instances: %d %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil || len(listResp.Items) == 0 {
		t.Fatalf("instances: err=%v body=%s", err, w.Body.String())
	}
	instanceID := listResp.Items[0].ID

	profileBody := `{"username":"FlowAPlayer"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/profiles", bytes.NewBufferString(profileBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create profile: %d %s", w.Code, w.Body.String())
	}
	var profileResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &profileResp); err != nil {
		t.Fatalf("profile json: %v", err)
	}

	launchBody := `{"instance_id":"` + instanceID + `","offline_profile_id":"` + profileResp.ID + `"}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/launcher/launch-requests", bytes.NewBufferString(launchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userTokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create launch: %d %s", w.Code, w.Body.String())
	}
	var launchResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &launchResp); err != nil {
		t.Fatalf("launch json: %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/launcher/launch-requests/pending", nil)
	req.Header.Set("Authorization", "Bearer "+deviceToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("pending: %d %s", w.Code, w.Body.String())
	}

	patchBody := `{"status":"running","pid":4242}`
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/launcher/launch-requests/"+launchResp.ID, bytes.NewBufferString(patchBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+deviceToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update launch: %d %s", w.Code, w.Body.String())
	}
}

func TestRouterServersFlow(t *testing.T) {
	r := setupRouter(t)

	regBody := `{"email":"servers-router@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}

	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("json: %v", err)
	}

	createBody, _ := json.Marshal(map[string]any{
		"name": "Router Dedicated",
		"ssh": map[string]any{
			"host":        "10.0.0.5",
			"username":    "ubuntu",
			"private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
		},
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create server: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list servers: %d %s", w.Code, w.Body.String())
	}

	var serversList struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &serversList); err != nil || len(serversList.Items) == 0 {
		t.Fatalf("servers list: err=%v body=%s", err, w.Body.String())
	}
	serverID := serversList.Items[0].ID

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID+"/deploy", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("deploy: %d %s", w.Code, w.Body.String())
	}

	// WebSocket auth uses query param; must not require Authorization header.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID+"/console?access_token="+tokens.AccessToken, nil)
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("console with query access_token should not be blocked by Bearer middleware")
	}
}

// Flow C (server admin): deploy → agent WSS connect → start command.
func TestRouterFlowC_AgentConnect(t *testing.T) {
	cap := &captureDeployer{}
	r := setupRouterWithDeploy(t, DeploySettings{
		PublicAPIURL:   "http://localhost:3000",
		DeployExecutor: cap,
	})
	httpSrv := httptest.NewServer(r)
	t.Cleanup(httpSrv.Close)

	regBody := `{"email":"flowc@test.com","password":"password123"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("json: %v", err)
	}

	createBody, _ := json.Marshal(map[string]any{
		"name": "Flow C Dedicated",
		"ssh": map[string]any{
			"host":        "10.0.0.8",
			"username":    "ubuntu",
			"private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
		},
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create server: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	var serversList struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &serversList); err != nil || len(serversList.Items) == 0 {
		t.Fatalf("servers list: err=%v body=%s", err, w.Body.String())
	}
	serverID := serversList.Items[0].ID

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID+"/deploy", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("deploy: %d %s", w.Code, w.Body.String())
	}
	if cap.token == "" || cap.serverID != serverID {
		t.Fatalf("expected captured agent token for server %s", serverID)
	}

	wsURL := strings.Replace(httpSrv.URL, "http://", "ws://", 1) + "/agent/v1/connect"
	header := http.Header{}
	header.Set("Authorization", "Bearer "+cap.token)
	header.Set("X-Agent-Hostname", "flowc-agent")
	header.Set("X-Agent-Version", "0.1.0")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("agent dial: %v", err)
	}
	defer conn.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("ws status: %d", resp.StatusCode)
	}
	// Drain agent commands so SendCommand writes do not block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
	_ = conn.WriteJSON(protocol.Envelope{
		V:    protocol.Version,
		Type: protocol.TypeEvtAgentHeartbeat,
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/v1/servers/"+serverID+"/start", nil)
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		r.ServeHTTP(w, req)
		if w.Code == http.StatusAccepted || w.Code == http.StatusOK {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("start: %d %s", w.Code, w.Body.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = conn.Close()
	<-done
}
