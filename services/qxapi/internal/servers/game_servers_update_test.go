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

func TestUpdateGameServer(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	gameServerID := "gs-update-1"
	rcon := "abc123"
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:           gameServerID,
		ServerID:     view.ID,
		Name:         "Old Name",
		ServerType:   "forge",
		MCVersion:    "1.20.1",
		Address:      strPtr("1.2.3.4"),
		Port:         25565,
		RconPassword: &rcon,
		Status:       models.GameServerStatusStopped,
		WorkDir:      "/opt/qxsystem/server/instances/gs-update-1",
		StartCommand: "/opt/qxsystem/server/instances/gs-update-1/run.sh",
	}).Error; err != nil {
		t.Fatalf("create game server row: %v", err)
	}

	var configurePayload protocol.ServerConfigurePayload
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
			if env.Type == protocol.TypeCmdServerConfigure {
				_ = json.Unmarshal(env.Payload, &configurePayload)
				resPayload, _ := json.Marshal(map[string]string{"status": "ok"})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerConfigure,
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

	newName := "New Name"
	newAddress := "203.0.113.8"
	newPort := 25570
	updated, err := svc.UpdateGameServer(ctx, "owner-1", view.ID, gameServerID, UpdateGameServerInput{
		Name:    &newName,
		Address: &newAddress,
		Port:    &newPort,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if updated.Name != newName || updated.Port != newPort {
		t.Fatalf("view: %+v", updated)
	}
	if updated.Address == nil || *updated.Address != newAddress {
		t.Fatalf("address: %+v", updated.Address)
	}
	if updated.RconPort != 35570 {
		t.Fatalf("rcon port: %d", updated.RconPort)
	}
	if configurePayload.Port != newPort || configurePayload.Name != newName {
		t.Fatalf("configure payload: %+v", configurePayload)
	}
}

func TestUpdateGameServerBusy(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	gameServerID := "gs-busy"
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         gameServerID,
		ServerID:   view.ID,
		Name:       "Busy",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Port:       25565,
		Status:     models.GameServerStatusInstalling,
	}).Error; err != nil {
		t.Fatal(err)
	}
	name := "X"
	_, err := svc.UpdateGameServer(ctx, "owner-1", view.ID, gameServerID, UpdateGameServerInput{Name: &name})
	if !errors.Is(err, ErrGameServerBusy) {
		t.Fatalf("expected busy: %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}
