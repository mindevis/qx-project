package servers

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestReinstallStoppedGameServerWipesBeforeInstall(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var commands []string
	gameServerID := "gs-reinstall-stopped"
	workDir := "/opt/qxsystem/server/instances/" + gameServerID

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
			mu.Lock()
			commands = append(commands, env.Type)
			mu.Unlock()
			switch env.Type {
			case protocol.TypeCmdServerWipe:
				resPayload, _ := json.Marshal(map[string]string{"status": "ok"})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerWipe,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdServerInstall:
				resPayload, _ := json.Marshal(protocol.ServerInstallResult{
					WorkDir: workDir,
					Command: workDir + "/run.sh",
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

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:            gameServerID,
		ServerID:      view.ID,
		Name:          "Reinstall Test",
		ServerType:    "forge",
		MCVersion:     "1.20.1",
		LoaderVersion: strPtr("47.4.20"),
		Port:          25565,
		Status:        models.GameServerStatusStopped,
		WorkDir:       workDir,
		StartCommand:  workDir + "/run.sh",
		StartArgsJSON: `["nogui"]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}

	reinstalled, err := svc.ReinstallGameServer(ctx, "owner-1", view.ID, gameServerID)
	if err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if reinstalled.Status != models.GameServerStatusInstalling {
		t.Fatalf("expected installing, got %s", reinstalled.Status)
	}

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	got := append([]string(nil), commands...)
	mu.Unlock()
	if len(got) < 2 || got[0] != protocol.TypeCmdServerWipe || got[1] != protocol.TypeCmdServerInstall {
		t.Fatalf("expected wipe then install, got %v", got)
	}
}

func TestReinstallRunningGameServerStopsThenWipesAndInstalls(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var commands []string
	gameServerID := "gs-reinstall-running"
	workDir := "/opt/qxsystem/server/instances/" + gameServerID

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
			mu.Lock()
			commands = append(commands, env.Type)
			mu.Unlock()
			switch env.Type {
			case protocol.TypeCmdServerStop:
				resPayload, _ := json.Marshal(protocol.ServerStopResult{ExitCode: 0})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerStop,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdServerWipe:
				resPayload, _ := json.Marshal(map[string]string{"status": "ok"})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerWipe,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdServerInstall:
				resPayload, _ := json.Marshal(protocol.ServerInstallResult{
					WorkDir: workDir,
					Command: workDir + "/run.sh",
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

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:            gameServerID,
		ServerID:      view.ID,
		Name:          "Running Reinstall",
		ServerType:    "forge",
		MCVersion:     "1.20.1",
		LoaderVersion: strPtr("47.4.20"),
		Port:          25565,
		Status:        models.GameServerStatusRunning,
		WorkDir:       workDir,
		StartCommand:  workDir + "/run.sh",
		StartArgsJSON: `["nogui"]`,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}

	reinstalled, err := svc.ReinstallGameServer(ctx, "owner-1", view.ID, gameServerID)
	if err != nil {
		t.Fatalf("reinstall: %v", err)
	}
	if reinstalled.Status != models.GameServerStatusInstalling {
		t.Fatalf("expected installing, got %s", reinstalled.Status)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	got := append([]string(nil), commands...)
	mu.Unlock()
	if len(got) < 3 ||
		got[0] != protocol.TypeCmdServerStop ||
		got[1] != protocol.TypeCmdServerWipe ||
		got[2] != protocol.TypeCmdServerInstall {
		t.Fatalf("expected stop, wipe, install; got %v", got)
	}
}
