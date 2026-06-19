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
	"github.com/qxproject/qx/services/qxapi/internal/servers"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestServerConsoleHandlerConnect(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	enc, _ := crypto.NewEncryptor(devSSHMasterKey())
	hub := agenthub.New(nil)
	svc := servers.NewService(db, tokens, enc, hub, servers.NoopDeployer{})
	hub.SetOnEvent(svc.OnAgentEvent)
	h := &ServerConsoleHandler{Servers: svc, Tokens: tokens}

	view, err := svc.Create(context.Background(), "owner-1", servers.CreateServerInput{
		Name: "Console Test",
		SSH:  servers.SSHInput{Host: "1.1.1.1", Username: "root", PrivateKey: testSSHKey},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	pair, err := tokens.IssueUserTokens("owner-1", "u@test.com")
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/servers/:id/console", h.Connect)

	server := httptest.NewServer(r)
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1) + "/servers/" + view.ID + "/console?access_token=" + pair.AccessToken

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	var status map[string]string
	if err := json.Unmarshal(data, &status); err != nil || status["type"] != "status" {
		t.Fatalf("status msg: %s err=%v", data, err)
	}

	payload, _ := json.Marshal(protocol.ConsoleOutputPayload{Stream: "stdout", Line: "test line"})
	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeEvtConsoleOutput, Payload: payload})

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var out agenthub.ConsolePanelMessage
	if err := json.Unmarshal(data, &out); err != nil || out.Line != "test line" {
		t.Fatalf("output: %s err=%v", data, err)
	}
}

func TestServerConsoleHandlerUnauthorized(t *testing.T) {
	h := &ServerConsoleHandler{
		Servers: nil,
		Tokens:  auth.NewTokenService("secret", time.Minute, time.Hour),
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/servers/x/console", nil)
	c.Params = gin.Params{{Key: "id", Value: "x"}}
	h.Connect(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth: %d", w.Code)
	}
}
