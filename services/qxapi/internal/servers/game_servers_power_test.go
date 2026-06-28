package servers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestGameServerStartStopRestart(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	server := httptestWSServer(t, func(conn *websocket.Conn) {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var env protocol.Envelope
			if json.Unmarshal(data, &env) != nil {
				continue
			}
			switch env.Type {
			case protocol.TypeCmdServerStart:
				resPayload, _ := json.Marshal(protocol.ServerStartResult{PID: 9001})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerStart,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdServerStop:
				resPayload, _ := json.Marshal(protocol.ServerStopResult{ExitCode: 0})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerStop,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			}
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

	gameServerID := "gs-power-1"
	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:            gameServerID,
		ServerID:      view.ID,
		Name:          "Power Test",
		ServerType:    "forge",
		MCVersion:     "1.20.1",
		Port:          25565,
		Status:        models.GameServerStatusStopped,
		WorkDir:       "/opt/qxsystem/server/instances/" + gameServerID,
		StartCommand:  "/opt/qxsystem/java/bin/java",
		StartArgsJSON: `["@user_jvm_args.txt","nogui"]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}

	started, err := svc.StartGameServer(ctx, "owner-1", view.ID, gameServerID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Status != models.GameServerStatusStarting {
		t.Fatalf("start status: %s", started.Status)
	}

	time.Sleep(100 * time.Millisecond)
	items, err := svc.ListGameServers(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if items[0].Status != models.GameServerStatusRunning {
		t.Fatalf("expected running after start, got %s", items[0].Status)
	}

	if _, err := svc.StopGameServer(ctx, "owner-1", view.ID, gameServerID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	items, err = svc.ListGameServers(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatalf("list after stop: %v", err)
	}
	if items[0].Status != models.GameServerStatusStopped {
		t.Fatalf("expected stopped, got %s", items[0].Status)
	}

	if _, err := svc.RestartGameServer(ctx, "owner-1", view.ID, gameServerID); err != nil {
		t.Fatalf("restart: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	items, err = svc.ListGameServers(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatalf("list after restart: %v", err)
	}
	if items[0].Status != models.GameServerStatusRunning {
		t.Fatalf("expected running after restart, got %s", items[0].Status)
	}
}

func TestStartGameServerNotInstalled(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	server := httptestWSServer(t, func(conn *websocket.Conn) {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
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

	now := time.Now().UTC()
	gameServerID := "gs-empty"
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:        gameServerID,
		ServerID:  view.ID,
		Name:      "Empty",
		ServerType: "forge",
		MCVersion: "1.20.1",
		Port:      25565,
		Status:    models.GameServerStatusStopped,
		WorkDir:   "/opt/qxsystem/server/instances/" + gameServerID,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = svc.StartGameServer(ctx, "owner-1", view.ID, gameServerID)
	if !errors.Is(err, ErrGameServerNotInstalled) {
		t.Fatalf("expected not installed: %v", err)
	}
}
