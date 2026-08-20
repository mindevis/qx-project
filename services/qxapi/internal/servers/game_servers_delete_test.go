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
	"gorm.io/gorm"
)

func TestDeleteGameServerWipesFilesThenDropsRow(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var wipes []protocol.GameServerWorkDirPayload
	gameServerID := "gs-delete-files"
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
			if env.Type != protocol.TypeCmdServerWipe {
				continue
			}
			var payload protocol.GameServerWorkDirPayload
			if json.Unmarshal(env.Payload, &payload) != nil {
				continue
			}
			mu.Lock()
			wipes = append(wipes, payload)
			mu.Unlock()
			resPayload, _ := json.Marshal(map[string]string{"status": "ok"})
			_ = conn.WriteJSON(protocol.Envelope{
				V:         protocol.Version,
				Type:      protocol.TypeResServerWipe,
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
	go hub.ReadLoop(hub.Register(view.ID, wsConn))

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         gameServerID,
		ServerID:   view.ID,
		Name:       "Delete Files",
		ServerType: "paper",
		MCVersion:  "26.2",
		Port:       25565,
		Status:     models.GameServerStatusStopped,
		WorkDir:    workDir,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("create game server: %v", err)
	}
	if err := svc.db.WithContext(ctx).Create(&models.GameServerInstanceBinding{
		ID:                 "bind-delete-files",
		UserID:             "owner-1",
		GameServerID:       gameServerID,
		LauncherInstanceID: "inst-delete-files",
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatalf("create binding: %v", err)
	}
	pid := 4242
	if err := svc.saveConfig(ctx, view.ID, ServerConfig{
		McPID:              &pid,
		ActiveGameServerID: gameServerID,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	if err := svc.DeleteGameServer(ctx, "owner-1", view.ID, gameServerID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	mu.Lock()
	got := append([]protocol.GameServerWorkDirPayload(nil), wipes...)
	mu.Unlock()
	if len(got) != 1 || got[0].WorkDir != workDir || !got[0].Remove {
		t.Fatalf("expected remove wipe of %s, got %+v", workDir, got)
	}

	var stored models.GameServer
	if err := svc.db.WithContext(ctx).Where("id = ?", gameServerID).First(&stored).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("game server row should be gone: err=%v", err)
	}
	var binding models.GameServerInstanceBinding
	if err := svc.db.WithContext(ctx).Where("id = ?", "bind-delete-files").First(&binding).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("binding should be gone: err=%v", err)
	}
	serverRow, err := svc.getByID(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := parseConfig(serverRow.ConfigJSON)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveGameServerID != "" || cfg.McPID != nil {
		t.Fatalf("active process should be cleared: %+v", cfg)
	}
}

func TestDeleteGameServerRequiresAgentWhenFilesExist(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	now := time.Now().UTC()
	gameServerID := "gs-delete-offline"
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         gameServerID,
		ServerID:   view.ID,
		Name:       "Offline Delete",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Port:       25565,
		Status:     models.GameServerStatusStopped,
		WorkDir:    "/opt/qxsystem/server/instances/" + gameServerID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteGameServer(ctx, "owner-1", view.ID, gameServerID); !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("expected agent offline, got %v", err)
	}
	var stored models.GameServer
	if err := svc.db.WithContext(ctx).Where("id = ?", gameServerID).First(&stored).Error; err != nil {
		t.Fatalf("row must remain until files are removed: %v", err)
	}
}

func TestDeleteVPSRemovesGameServerFilesWhenAgentOnline(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var removed []string
	gameServerID := "gs-vps-delete"
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
			if env.Type != protocol.TypeCmdServerWipe {
				continue
			}
			var payload protocol.GameServerWorkDirPayload
			_ = json.Unmarshal(env.Payload, &payload)
			mu.Lock()
			removed = append(removed, payload.WorkDir)
			mu.Unlock()
			resPayload, _ := json.Marshal(map[string]string{"status": "ok"})
			_ = conn.WriteJSON(protocol.Envelope{
				V:         protocol.Version,
				Type:      protocol.TypeResServerWipe,
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
	go hub.ReadLoop(hub.Register(view.ID, wsConn))

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         gameServerID,
		ServerID:   view.ID,
		Name:       "VPS Delete",
		ServerType: "forge",
		MCVersion:  "1.20.1",
		Port:       25565,
		Status:     models.GameServerStatusStopped,
		WorkDir:    workDir,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("delete vps: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), removed...)
	mu.Unlock()
	if len(got) != 1 || got[0] != workDir {
		t.Fatalf("expected wipe of %s, got %v", workDir, got)
	}
	if err := svc.db.WithContext(ctx).Where("id = ?", gameServerID).First(&models.GameServer{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("game server should be gone: %v", err)
	}
}
