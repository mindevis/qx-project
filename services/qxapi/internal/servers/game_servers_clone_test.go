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

func TestCloneGameServerCopiesWorkDir(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	var mu sync.Mutex
	var copies []protocol.GameServerCopyPayload
	srcID := "gs-clone-src"
	srcWorkDir := "/opt/qxsystem/server/instances/" + srcID
	rcon := "src-rcon"

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
			case protocol.TypeCmdServerCopy:
				var payload protocol.GameServerCopyPayload
				if json.Unmarshal(env.Payload, &payload) != nil {
					continue
				}
				mu.Lock()
				copies = append(copies, payload)
				mu.Unlock()
				resPayload, _ := json.Marshal(map[string]string{"status": "ok"})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResServerCopy,
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
	go hub.ReadLoop(hub.Register(view.ID, wsConn))

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:                    srcID,
		ServerID:              view.ID,
		Name:                  "Paper RPG",
		ServerType:            "paper",
		MCVersion:             "1.21.4",
		Address:               strPtr("play.example.com"),
		Port:                  25565,
		RconPassword:          &rcon,
		Status:                models.GameServerStatusStopped,
		WorkDir:               srcWorkDir,
		StartCommand:          srcWorkDir + "/run.sh",
		StartArgsJSON:         `["nogui"]`,
		JarPath:               srcWorkDir + "/server.jar",
		ShowInMonitoring:      true,
		MonitoringDescription: "public",
		ContentResources: models.InstanceResourceList{
			{Source: "modrinth", ProjectID: "viaversion", Filename: "via.jar", ResourceType: "plugin"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create source: %v", err)
	}

	cloned, err := svc.CloneGameServer(ctx, "owner-1", view.ID, srcID)
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	if cloned.ID == srcID {
		t.Fatal("clone should get a new id")
	}
	if cloned.Name != "Paper RPG (copy)" {
		t.Fatalf("name: %q", cloned.Name)
	}
	if cloned.Port == 25565 {
		t.Fatalf("expected a free port, got %d", cloned.Port)
	}
	if cloned.Status != models.GameServerStatusStopped {
		t.Fatalf("status: %s", cloned.Status)
	}
	if cloned.ShowInMonitoring {
		t.Fatal("clone must not be listed in monitoring")
	}
	if cloned.RconPassword == nil || *cloned.RconPassword == rcon {
		t.Fatal("clone should get a new rcon password")
	}
	if cloned.MonitoringDescription != "public" {
		t.Fatalf("description: %q", cloned.MonitoringDescription)
	}

	var dest models.GameServer
	if err := svc.db.WithContext(ctx).Where("id = ?", cloned.ID).First(&dest).Error; err != nil {
		t.Fatalf("reload dest: %v", err)
	}
	if dest.WorkDir != "/opt/qxsystem/server/instances/"+cloned.ID {
		t.Fatalf("work dir: %s", dest.WorkDir)
	}
	if dest.StartCommand != dest.WorkDir+"/run.sh" {
		t.Fatalf("start command: %s", dest.StartCommand)
	}
	if dest.JarPath != dest.WorkDir+"/server.jar" {
		t.Fatalf("jar path: %s", dest.JarPath)
	}
	if len(dest.ContentResources) != 1 || dest.ContentResources[0].Filename != "via.jar" {
		t.Fatalf("content: %+v", dest.ContentResources)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(copies)
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	got := append([]protocol.GameServerCopyPayload(nil), copies...)
	mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("expected 1 copy command, got %d", len(got))
	}
	if got[0].SourceWorkDir != srcWorkDir || got[0].DestWorkDir != dest.WorkDir {
		t.Fatalf("copy payload: %+v", got[0])
	}
}

func TestCloneGameServerBusy(t *testing.T) {
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
	go hub.ReadLoop(hub.Register(view.ID, wsConn))

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         "gs-busy",
		ServerID:   view.ID,
		Name:       "Busy",
		ServerType: "paper",
		MCVersion:  "1.21",
		Port:       25565,
		Status:     models.GameServerStatusInstalling,
		WorkDir:    "/opt/qxsystem/server/instances/gs-busy",
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.CloneGameServer(ctx, "owner-1", view.ID, "gs-busy")
	if !errors.Is(err, ErrGameServerBusy) {
		t.Fatalf("expected busy, got %v", err)
	}
}

func TestCloneGameServerNotInstalled(t *testing.T) {
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
	go hub.ReadLoop(hub.Register(view.ID, wsConn))

	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.GameServer{
		ID:         "gs-empty",
		ServerID:   view.ID,
		Name:       "Empty",
		ServerType: "vanilla",
		MCVersion:  "1.21",
		Port:       25565,
		Status:     models.GameServerStatusStopped,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.CloneGameServer(ctx, "owner-1", view.ID, "gs-empty")
	if !errors.Is(err, ErrGameServerNotInstalled) {
		t.Fatalf("expected not installed, got %v", err)
	}
}
