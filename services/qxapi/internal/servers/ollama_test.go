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

func TestGetOllamaNotInstalled(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	got, err := svc.GetOllama(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.OllamaStatusNotInstalled {
		t.Fatalf("status: %s", got.Status)
	}
	if got.Models == nil {
		t.Fatal("expected empty models slice")
	}
}

func TestInstallOllamaRequiresAgent(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.InstallOllama(ctx, "owner-1", view.ID); !errors.Is(err, ErrNotDeployed) {
		t.Fatalf("expected not deployed: %v", err)
	}
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.InstallOllama(ctx, "owner-1", view.ID); !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("expected agent offline: %v", err)
	}
}

func TestOllamaInstallStartPull(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatal(err)
	}

	running := false
	var runningMu sync.Mutex
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
			case protocol.TypeCmdOllamaInstall:
				resPayload, _ := json.Marshal(protocol.OllamaInstallResult{
					Version:    "0.9.0",
					BinPath:    "/opt/qxsystem/ollama/bin/ollama",
					RootDir:    "/opt/qxsystem/ollama",
					ModelsDir:  "/opt/qxsystem/ollama/models",
					ListenAddr: "127.0.0.1:11434",
				})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResOllamaInstall,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdOllamaStart:
				runningMu.Lock()
				running = true
				runningMu.Unlock()
				resPayload, _ := json.Marshal(ollamaRunningStatus(true))
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResOllamaStart,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdOllamaStatus:
				runningMu.Lock()
				isRunning := running
				runningMu.Unlock()
				resPayload, _ := json.Marshal(ollamaRunningStatus(isRunning))
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResOllamaStatus,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdOllamaModelList:
				resPayload, _ := json.Marshal(protocol.OllamaModelListResult{
					Models: []protocol.OllamaModel{{Name: "llama3.2:latest", Size: 12}},
				})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResOllamaModelList,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdOllamaModelPull:
				resPayload, _ := json.Marshal(map[string]string{"status": "ok", "name": "phi4"})
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResOllamaModelPull,
					RequestID: env.RequestID,
					Payload:   resPayload,
				})
			case protocol.TypeCmdOllamaStop:
				runningMu.Lock()
				running = false
				runningMu.Unlock()
				_ = conn.WriteJSON(protocol.Envelope{
					V:         protocol.Version,
					Type:      protocol.TypeResOllamaStop,
					RequestID: env.RequestID,
					Payload:   []byte(`{"status":"ok"}`),
				})
			}
		}
	})
	defer server.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer wsConn.Close()
	agentConn := hub.Register(view.ID, wsConn)
	go hub.ReadLoop(agentConn)

	installed, err := svc.InstallOllama(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Status != models.OllamaStatusInstalling && installed.Status != models.OllamaStatusStarting && installed.Status != models.OllamaStatusRunning {
		t.Fatalf("install status: %s", installed.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
	var got *OllamaView
	for time.Now().Before(deadline) {
		got, err = svc.GetOllama(ctx, "owner-1", view.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == models.OllamaStatusRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != models.OllamaStatusRunning {
		t.Fatalf("expected running, got %+v", got)
	}
	if len(got.Models) != 1 || got.Models[0].Name != "llama3.2:latest" {
		t.Fatalf("models: %+v", got.Models)
	}

	pulled, err := svc.PullOllamaModel(ctx, "owner-1", view.ID, "phi4")
	if err != nil {
		t.Fatal(err)
	}
	if pulled.Status != models.OllamaStatusPulling && pulled.Status != models.OllamaStatusRunning {
		t.Fatalf("pull status: %s", pulled.Status)
	}
	time.Sleep(80 * time.Millisecond)
	got, err = svc.GetOllama(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.OllamaStatusRunning {
		t.Fatalf("after pull: %s", got.Status)
	}

	stopped, err := svc.StopOllama(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != models.OllamaStatusInstalled {
		t.Fatalf("stop status: %s", stopped.Status)
	}
}

func TestPullOllamaModelRejectsInvalidName(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.PullOllamaModel(ctx, "owner-1", view.ID, "../x"); !errors.Is(err, ErrOllamaInvalidModel) {
		t.Fatalf("expected invalid: %v", err)
	}
}

func ollamaRunningStatus(running bool) protocol.OllamaStatusResult {
	return protocol.OllamaStatusResult{
		Installed:  true,
		Running:    running,
		Version:    "0.9.0",
		BinPath:    "/opt/qxsystem/ollama/bin/ollama",
		RootDir:    "/opt/qxsystem/ollama",
		ListenAddr: "127.0.0.1:11434",
	}
}
