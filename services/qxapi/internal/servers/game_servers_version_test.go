package servers

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestChangeStoppedGameServerVersionInstallsWithoutWipe(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var commands []string
	var install protocol.ServerInstallPayload
	gameServerID := "gs-version-stopped"
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
			if env.Type == protocol.TypeCmdServerInstall {
				_ = json.Unmarshal(env.Payload, &install)
			}
			mu.Unlock()
			switch env.Type {
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
		Name:          "Version Test",
		ServerType:    "paper",
		MCVersion:     "1.20.1",
		LoaderVersion: strPtr("100"),
		Port:          25565,
		Status:        models.GameServerStatusStopped,
		WorkDir:       workDir,
		StartCommand:  workDir + "/run.sh",
		StartArgsJSON: `["nogui"]`,
		ContentResources: models.InstanceResourceList{
			{Source: "modrinth", ProjectID: "spark", Filename: "spark.jar", ResourceType: "plugin"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}

	changed, err := svc.ChangeGameServerVersion(ctx, "owner-1", view.ID, gameServerID, ChangeGameServerVersionInput{
		MCVersion:     "1.21.1",
		LoaderVersion: "210",
	})
	if err != nil {
		t.Fatalf("change version: %v", err)
	}
	if changed.Status != models.GameServerStatusInstalling {
		t.Fatalf("expected installing, got %s", changed.Status)
	}
	if changed.MCVersion != "1.21.1" {
		t.Fatalf("mc version: %s", changed.MCVersion)
	}
	if changed.LoaderVersion == nil || *changed.LoaderVersion != "210" {
		t.Fatalf("loader version: %v", changed.LoaderVersion)
	}

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	got := append([]string(nil), commands...)
	gotInstall := install
	mu.Unlock()
	if len(got) < 1 || got[0] != protocol.TypeCmdServerInstall {
		t.Fatalf("expected install without wipe, got %v", got)
	}
	for _, cmd := range got {
		if cmd == protocol.TypeCmdServerWipe {
			t.Fatalf("version change must not wipe work dir, got %v", got)
		}
	}
	if gotInstall.MCVersion != "1.21.1" || gotInstall.LoaderVersion != "210" {
		t.Fatalf("install payload versions: %+v", gotInstall)
	}

	var stored models.GameServer
	if err := svc.db.WithContext(ctx).Where("id = ?", gameServerID).First(&stored).Error; err != nil {
		t.Fatalf("reload game server: %v", err)
	}
	if len(stored.ContentResources) != 1 {
		t.Fatalf("version change must keep catalog installs, got %+v", stored.ContentResources)
	}
}

func TestChangeRunningGameServerVersionStopsThenInstallsWithoutWipe(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var commands []string
	gameServerID := "gs-version-running"
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
		Name:          "Running Version",
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

	changed, err := svc.ChangeGameServerVersion(ctx, "owner-1", view.ID, gameServerID, ChangeGameServerVersionInput{
		MCVersion:     "1.20.1",
		LoaderVersion: "47.3.0",
	})
	if err != nil {
		t.Fatalf("change version: %v", err)
	}
	if changed.Status != models.GameServerStatusInstalling {
		t.Fatalf("expected installing, got %s", changed.Status)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	got := append([]string(nil), commands...)
	mu.Unlock()
	if len(got) < 2 || got[0] != protocol.TypeCmdServerStop || got[1] != protocol.TypeCmdServerInstall {
		t.Fatalf("expected stop then install, got %v", got)
	}
	for _, cmd := range got {
		if cmd == protocol.TypeCmdServerWipe {
			t.Fatalf("version change must not wipe work dir, got %v", got)
		}
	}
}

func TestChangeGameServerVersionValidationAndBusy(t *testing.T) {
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
	hub.Register(view.ID, wsConn)

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:            "gs-version-validate",
		ServerID:      view.ID,
		Name:          "Validate",
		ServerType:    "paper",
		MCVersion:     "1.21",
		LoaderVersion: strPtr("12"),
		Port:          25565,
		Status:        models.GameServerStatusStopped,
		WorkDir:       "/opt/qxsystem/server/instances/gs-version-validate",
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}

	if _, err := svc.ChangeGameServerVersion(ctx, "owner-1", view.ID, "gs-version-validate", ChangeGameServerVersionInput{
		MCVersion: "1.21.1",
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for missing loader, got %v", err)
	}

	unchanged, err := svc.ChangeGameServerVersion(ctx, "owner-1", view.ID, "gs-version-validate", ChangeGameServerVersionInput{
		MCVersion:     "1.21",
		LoaderVersion: "12",
	})
	if err != nil {
		t.Fatalf("unchanged versions: %v", err)
	}
	if unchanged.Status != models.GameServerStatusStopped {
		t.Fatalf("unchanged versions must not start install, got %s", unchanged.Status)
	}

	if err := svc.db.WithContext(ctx).Model(&models.GameServer{}).Where("id = ?", "gs-version-validate").
		Update("status", models.GameServerStatusInstalling).Error; err != nil {
		t.Fatalf("set installing: %v", err)
	}
	if _, err := svc.ChangeGameServerVersion(ctx, "owner-1", view.ID, "gs-version-validate", ChangeGameServerVersionInput{
		MCVersion:     "1.20.4",
		LoaderVersion: "8",
	}); !errors.Is(err, ErrGameServerBusy) {
		t.Fatalf("expected busy, got %v", err)
	}
}
