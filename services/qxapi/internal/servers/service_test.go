package servers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

const testSSHKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDExampleKeyForTestsOnlyDoNotUseInProduction==
-----END OPENSSH PRIVATE KEY-----`

func devKey() string {
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}

func newServersService(t *testing.T) (*Service, *auth.TokenService, *agenthub.Hub) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("test-secret", time.Minute, time.Hour)
	enc, err := crypto.NewEncryptor(devKey())
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}
	hub := agenthub.New(nil)
	svc := NewService(db, tokens, enc, hub, NoopDeployer{})
	hub.SetOnEvent(svc.OnAgentEvent)
	return svc, tokens, hub
}

func createTestServer(t *testing.T, svc *Service, ownerID string) *ServerView {
	t.Helper()
	view, err := svc.Create(context.Background(), ownerID, CreateServerInput{
		Name: "Test Server",
		SSH: SSHInput{
			Host:       "10.0.0.1",
			Port:       22,
			Username:   "root",
			PrivateKey: testSSHKey,
		},
		Config: ServerConfig{JarPath: "/opt/qxsystem/server/server.jar"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return view
}

func TestCreateListGetDelete(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	owner := "owner-1"

	view := createTestServer(t, svc, owner)
	if view.Name != "Test Server" || view.Status != models.ServerStatusPending {
		t.Fatalf("view: %+v", view)
	}
	if view.AgentDeployed {
		t.Fatal("expected agent_deployed false before deploy")
	}

	items, err := svc.List(ctx, owner)
	if err != nil || len(items) != 1 {
		t.Fatalf("list: err=%v len=%d", err, len(items))
	}

	got, err := svc.Get(ctx, owner, view.ID)
	if err != nil || got.ID != view.ID {
		t.Fatalf("get: err=%v got=%+v", err, got)
	}

	if err := svc.Delete(ctx, owner, view.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(ctx, owner, view.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete again: %v", err)
	}
}

func TestCreateValidation(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "owner", CreateServerInput{Name: "  "})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name: %v", err)
	}

	_, err = svc.Create(ctx, "owner", CreateServerInput{
		Name: "Srv",
		SSH:  SSHInput{Host: "h", Username: "u", PrivateKey: ""},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("empty key: %v", err)
	}
}

func TestForbiddenAccess(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	_, err := svc.Get(ctx, "other", view.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("get forbidden: %v", err)
	}

	if err := svc.Delete(ctx, "other", view.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("delete forbidden: %v", err)
	}
}

func TestDeployAndAgentToken(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	deployed, err := svc.Deploy(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if deployed.View.AgentDeployed != true {
		t.Fatal("expected agent_deployed true after deploy")
	}
	if deployed.View.Status != models.ServerStatusOffline {
		t.Fatalf("status: %s", deployed.View.Status)
	}
	if deployed.AgentToken == "" {
		t.Fatal("expected agent token")
	}

	token := deployed.AgentToken
	hash := auth.HashToken(token)
	if err := svc.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", view.ID).Update("agent_token_hash", hash).Error; err != nil {
		t.Fatalf("set hash: %v", err)
	}
	if err := svc.ValidateAgentToken(ctx, view.ID, token); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := svc.ValidateAgentToken(ctx, view.ID, "wrong"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong token: %v", err)
	}
}

func TestLifecycleAgentOffline(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	if err := svc.Start(ctx, "owner-1", view.ID); !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("start offline: %v", err)
	}
	if err := svc.Stop(ctx, "owner-1", view.ID); !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("stop offline: %v", err)
	}
}

func TestStartNotDeployed(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	if err := svc.Start(ctx, "owner-1", view.ID); !errors.Is(err, ErrNotDeployed) {
		t.Fatalf("start not deployed: %v", err)
	}
}

func TestAgentConnectDisconnect(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.Agent{
		ID:        "agent-test-1",
		ServerID:  view.ID,
		OS:        "linux",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create agent row: %v", err)
	}

	if err := svc.AgentConnected(ctx, view.ID, "dedicated server-1", "0.1.0"); err != nil {
		t.Fatalf("connected: %v", err)
	}
	got, err := svc.Get(ctx, "owner-1", view.ID)
	if err != nil || got.Status != models.ServerStatusPending {
		t.Fatalf("agent connect should not mark mc online: err=%v status=%s", err, got.Status)
	}
	if got.LastSeenAt == nil {
		t.Fatal("expected last_seen_at after agent connect")
	}
	if got.AgentVersion == nil || *got.AgentVersion != "0.1.0" {
		t.Fatalf("expected agent_version 0.1.0, got %v", got.AgentVersion)
	}

	_ = svc.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", view.ID).Updates(map[string]any{
		"status":      models.ServerStatusOnline,
		"config_json": `{"mc_pid":4242,"active_game_server_id":"gs-1"}`,
	}).Error
	if err := svc.AgentConnected(ctx, view.ID, "dedicated server-1", "0.1.0"); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	got, err = svc.Get(ctx, "owner-1", view.ID)
	if err != nil || got.Status != models.ServerStatusOnline || !got.MinecraftRunning {
		t.Fatalf("agent reconnect should keep running mc until status event: status=%s mc=%v", got.Status, got.MinecraftRunning)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{
		Type:    protocol.TypeEvtServerStatus,
		Payload: []byte(`{"status":"stopped","game_server_id":"gs-1"}`),
	})
	got, err = svc.Get(ctx, "owner-1", view.ID)
	if err != nil || got.Status != models.ServerStatusOffline || got.MinecraftRunning {
		t.Fatalf("stopped status event should clear mc online: status=%s mc=%v", got.Status, got.MinecraftRunning)
	}

	if err := svc.AgentDisconnected(ctx, view.ID); err != nil {
		t.Fatalf("disconnected: %v", err)
	}
	got, err = svc.Get(ctx, "owner-1", view.ID)
	if err != nil || got.Status != models.ServerStatusOffline {
		t.Fatalf("offline: err=%v status=%s", err, got.Status)
	}

	now = time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         "gs-keep-running",
		ServerID:   view.ID,
		Name:       "Keep Running",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Status:     models.GameServerStatusRunning,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_ = svc.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", view.ID).Updates(map[string]any{
		"status":      models.ServerStatusOnline,
		"config_json": `{"mc_pid":4242,"active_game_server_id":"gs-keep-running"}`,
	}).Error
	if err := svc.AgentDisconnected(ctx, view.ID); err != nil {
		t.Fatalf("disconnected while running: %v", err)
	}
	var kept models.GameServer
	if err := svc.db.WithContext(ctx).Where("id = ?", "gs-keep-running").First(&kept).Error; err != nil {
		t.Fatal(err)
	}
	if kept.Status != models.GameServerStatusRunning {
		t.Fatalf("agent disconnect must not stop the game server: %s", kept.Status)
	}
}

func TestOnAgentEvent(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeEvtAgentHeartbeat})
	got, _ := svc.Get(ctx, "owner-1", view.ID)
	if got.Status != models.ServerStatusPending || got.LastSeenAt == nil {
		t.Fatalf("heartbeat should only touch last_seen_at: %+v", got)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeResServerStart, Payload: []byte(`{"pid":4242}`)})
	got, _ = svc.Get(ctx, "owner-1", view.ID)
	if got.Status != models.ServerStatusOnline {
		t.Fatalf("start res: %s", got.Status)
	}
	if !got.MinecraftRunning || got.Config.McPID == nil || *got.Config.McPID != 4242 {
		t.Fatalf("expected mc pid persisted: %+v", got)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeResServerStart, Payload: []byte(`{"error":"java not found"}`)})
	got, _ = svc.Get(ctx, "owner-1", view.ID)
	if got.Status != models.ServerStatusOffline {
		t.Fatalf("start error should leave host offline, not error: %s", got.Status)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeResServerStart, Payload: []byte(`{}`)})
	got, _ = svc.Get(ctx, "owner-1", view.ID)
	if got.Status != models.ServerStatusOffline {
		t.Fatalf("start empty: %s", got.Status)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeResServerStop})
	got, _ = svc.Get(ctx, "owner-1", view.ID)
	if got.Status != models.ServerStatusOffline {
		t.Fatalf("stop res: %s", got.Status)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{Type: protocol.TypeEvtConsoleOutput})
}

func TestApplyServerStatusEventCrash(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	now := time.Now().UTC()
	gameServerID := "gs-crash"
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         gameServerID,
		ServerID:   view.ID,
		Name:       "Crash Test",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Status:     models.GameServerStatusRunning,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	svc.OnAgentEvent(view.ID, protocol.Envelope{
		Type: protocol.TypeEvtServerStatus,
		Payload: []byte(`{
			"status":"crashed",
			"game_server_id":"gs-crash",
			"message":"minecraft server exited unexpectedly (code 1)\nfatal boot error"
		}`),
	})

	var item models.GameServer
	if err := svc.db.WithContext(ctx).Where("id = ?", gameServerID).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.Status != models.GameServerStatusError {
		t.Fatalf("status: %s", item.Status)
	}
	if item.LastError == "" {
		t.Fatal("expected last_error")
	}
}

func TestStaleOnlineWithoutMcPID(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if err := svc.db.WithContext(ctx).Model(&models.Server{}).Where("id = ?", view.ID).
		Update("status", models.ServerStatusOnline).Error; err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.ServerStatusOffline || got.MinecraftRunning {
		t.Fatalf("stale online should normalize: status=%s mc=%v", got.Status, got.MinecraftRunning)
	}
}

func TestSendConsoleInput(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	server := httptestWSServer(t, func(conn *websocket.Conn) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) == nil && env.Type == protocol.TypeCmdConsoleInput {
			payload, _ := json.Marshal(protocol.ConsoleOutputPayload{Stream: "stdout", Line: "ok"})
			_ = conn.WriteJSON(protocol.Envelope{Type: protocol.TypeEvtConsoleOutput, Payload: payload})
		}
	})
	defer server.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer wsConn.Close()
	agentConn := hub.Register(view.ID, wsConn)
	go hub.ReadLoop(agentConn)

	if err := svc.SendConsoleInput(ctx, "owner-1", view.ID, "say hi", ""); err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestAttachConsole(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	received := make(chan protocol.ConsoleAttachPayload, 1)
	server := httptestWSServer(t, func(conn *websocket.Conn) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) != nil || env.Type != protocol.TypeCmdConsoleAttach {
			return
		}
		var payload protocol.ConsoleAttachPayload
		if json.Unmarshal(env.Payload, &payload) == nil {
			received <- payload
		}
	})
	defer server.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer wsConn.Close()
	agentConn := hub.Register(view.ID, wsConn)
	go hub.ReadLoop(agentConn)

	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         "gs-attach",
		ServerID:   view.ID,
		Name:       "Test",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Status:     models.GameServerStatusRunning,
		WorkDir:    "/opt/qxsystem/server/instances/gs-attach",
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}

	if err := svc.AttachConsole(ctx, "owner-1", view.ID, "gs-attach"); err != nil {
		t.Fatalf("attach: %v", err)
	}

	select {
	case payload := <-received:
		if payload.WorkDir != "/opt/qxsystem/server/instances/gs-attach" || payload.GameServerID != "gs-attach" {
			t.Fatalf("payload: %+v", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for attach command")
	}
}

func TestSendCommandWhenOnline(t *testing.T) {
	svc, tokens, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	server := httptestWSServer(t, func(conn *websocket.Conn) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) != nil {
			return
		}
		if env.Type == protocol.TypeCmdServerStart {
			resPayload, _ := json.Marshal(protocol.ServerStartResult{PID: 1234})
			_ = conn.WriteJSON(protocol.Envelope{
				V:         protocol.Version,
				Type:      protocol.TypeResServerStart,
				RequestID: env.RequestID,
				Payload:   resPayload,
			})
		}
	})
	defer server.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer wsConn.Close()

	agentConn := hub.Register(view.ID, wsConn)
	go hub.ReadLoop(agentConn)

	if err := svc.Start(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("start: %v", err)
	}

	_ = tokens
	time.Sleep(50 * time.Millisecond)
	got, _ := svc.Get(ctx, "owner-1", view.ID)
	if got.Status != models.ServerStatusStarting && got.Status != models.ServerStatusOnline {
		t.Fatalf("status after start: %s", got.Status)
	}
}

func TestSlugifyAndParseConfig(t *testing.T) {
	if slugify("My Cool Server!!!") != "my-cool-server" {
		t.Fatalf("slugify")
	}
	if slugify("   ") != "server" {
		t.Fatalf("empty slug")
	}

	cfg, err := parseConfig(`{"jar_path":"/jar","jvm_args":["-Xmx1G"]}`)
	if err != nil || cfg.JarPath != "/jar" || len(cfg.JVMArgs) != 1 {
		t.Fatalf("parse: err=%v cfg=%+v", err, cfg)
	}
	_, err = parseConfig("{bad")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestDeployFailure(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	failSvc := NewService(svc.db, svc.tokens, svc.enc, svc.hub, deployFail{})
	_, err := failSvc.Deploy(ctx, "owner-1", view.ID)
	if err == nil {
		t.Fatal("expected deploy error")
	}
}

type deployFail struct{}

func (deployFail) Deploy(context.Context, string, models.SSHCredential, string) error {
	return errors.New("ssh failed")
}

func httptestWSServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(conn)
	}))
}

func wsURL(httpURL string) string {
	return strings.Replace(httpURL, "http://", "ws://", 1)
}
