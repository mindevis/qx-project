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

func TestCreateGameServerProvisionFlow(t *testing.T) {
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
			case protocol.TypeCmdServerInstall:
				resPayload, _ := json.Marshal(protocol.ServerInstallResult{
					WorkDir: "/opt/qx/server/instances/gs-1",
					Command: "/opt/qx/server/instances/gs-1/run.sh",
					Args:    []string{"nogui"},
				})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerInstall,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdServerStart:
				resPayload, _ := json.Marshal(protocol.ServerStartResult{PID: 9001})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerStart,
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

	created, err := svc.CreateGameServer(ctx, "owner-1", view.ID, CreateGameServerInput{
		Name:          "Forge Test",
		ServerType:    "forge",
		MCVersion:     "1.20.1",
		LoaderVersion: "47.4.20",
		Address:       "1.2.3.4",
		Port:          25565,
	})
	if err != nil {
		t.Fatalf("create game server: %v", err)
	}
	if created.Status != models.GameServerStatusInstalling {
		t.Fatalf("initial status: %s", created.Status)
	}
	if created.RconPassword == nil || *created.RconPassword == "" {
		t.Fatal("expected rcon password on create")
	}
	if created.RconPort != 35565 {
		t.Fatalf("rcon port: %d", created.RconPort)
	}

	time.Sleep(100 * time.Millisecond)

	items, err := svc.ListGameServers(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 game server, got %d", len(items))
	}
	if items[0].Status != models.GameServerStatusRunning {
		t.Fatalf("expected running, got %s", items[0].Status)
	}

	got, err := svc.Get(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatalf("get vps: %v", err)
	}
	if got.ServerType != "forge" || got.Status != models.ServerStatusOnline {
		t.Fatalf("vps after provision: type=%s status=%s", got.ServerType, got.Status)
	}
	if got.Config.Command == "" || got.Config.WorkDir == "" {
		t.Fatalf("expected start config: %+v", got.Config)
	}
}

func TestCreateGameServerAgentOffline(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	_, err := svc.CreateGameServer(ctx, "owner-1", view.ID, CreateGameServerInput{
		Name:          "Forge Test",
		ServerType:    "forge",
		MCVersion:     "1.20.1",
		LoaderVersion: "47.4.20",
		Port:          25565,
	})
	if !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("expected agent offline: %v", err)
	}
}

func TestCreateGameServerValidation(t *testing.T) {
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

	_, err = svc.CreateGameServer(ctx, "owner-1", view.ID, CreateGameServerInput{
		Name:       "Forge Test",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Port:       25565,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation: %v", err)
	}
}
