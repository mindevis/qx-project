package api

import (
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
	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestAgentWSHandlerConnect(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	enc, _ := crypto.NewEncryptor(devSSHMasterKey())
	hub := agenthub.New(nil)
	svc := servers.NewService(db, tokens, enc, hub, servers.NoopDeployer{})
	hub.SetOnEvent(svc.OnAgentEvent)
	h := &AgentWSHandler{Hub: hub, Tokens: tokens, Servers: svc}

	view, err := svc.Create(context.Background(), "owner-1", servers.CreateServerInput{
		Name: "WS Test",
		SSH: servers.SSHInput{
			Host: "1.1.1.1", Username: "root", PrivateKey: testSSHKey,
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Deploy(context.Background(), "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	token, err := tokens.IssueAgentToken(view.ID, time.Hour)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	hash := auth.HashToken(token)
	if err := db.Model(&models.Server{}).Where("id = ?", view.ID).Update("agent_token_hash", hash).Error; err != nil {
		t.Fatalf("hash: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/agent/v1/connect", h.Connect)

	server := httptest.NewServer(r)
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/agent/v1/connect"

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	header.Set("X-Agent-Hostname", "test-host")
	header.Set("X-Agent-Version", "0.1.0")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status: %d", resp.StatusCode)
	}

	payload, _ := json.Marshal(protocol.ServerStopPayload{Graceful: true})
	_ = conn.WriteJSON(protocol.Envelope{
		V:    protocol.Version,
		Type: protocol.TypeEvtAgentHeartbeat,
		Payload: payload,
	})
	_ = conn.Close()
}

func TestAgentWSHandlerUnauthorized(t *testing.T) {
	h := &AgentWSHandler{
		Hub:     agenthub.New(nil),
		Tokens:  auth.NewTokenService("secret", time.Minute, time.Hour),
		Servers: nil,
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/agent/v1/connect", nil)
	h.Connect(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth: %d", w.Code)
	}
}
