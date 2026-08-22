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

func TestGetMySQLNotInstalled(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	got, err := svc.GetMySQL(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.MySQLStatusNotInstalled {
		t.Fatalf("status: %s", got.Status)
	}
	if got.Databases == nil || got.Users == nil || len(got.PrivilegeCatalog) == 0 {
		t.Fatalf("expected empty collections: %+v", got)
	}
}

func TestInstallMySQLRequiresAgent(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	in := MySQLInstallInput{Engine: "mariadb", Version: "8.0", Method: "docker"}
	if _, err := svc.InstallMySQL(ctx, "owner-1", view.ID, in); !errors.Is(err, ErrNotDeployed) {
		t.Fatalf("expected not deployed: %v", err)
	}
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.InstallMySQL(ctx, "owner-1", view.ID, in); !errors.Is(err, ErrAgentOffline) {
		t.Fatalf("expected agent offline: %v", err)
	}
}

func TestInstallMySQLRejectsEngine(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.InstallMySQL(ctx, "owner-1", view.ID, MySQLInstallInput{
		Engine: "postgres", Version: "8.0", Method: "docker",
	}); !errors.Is(err, ErrMySQLInvalidEngine) {
		t.Fatalf("expected invalid engine: %v", err)
	}
}

func TestMySQLInstallStartDatabaseUser(t *testing.T) {
	svc, _, hub := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	if _, err := svc.Deploy(ctx, "owner-1", view.ID); err != nil {
		t.Fatal(err)
	}

	running := false
	var mu sync.Mutex
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
			case protocol.TypeCmdMySQLInstall:
				resPayload, _ := json.Marshal(protocol.MySQLInstallResult{
					Engine:   "mariadb",
					Version:  "8.0",
					Method:   "docker",
					BindAddr: "127.0.0.1",
					Port:     3306,
					Image:    "mariadb:11.4",
				})
				_ = conn.WriteJSON(protocol.Envelope{
					V: protocol.Version, Type: protocol.TypeResMySQLInstall, RequestID: env.RequestID, Payload: resPayload,
				})
			case protocol.TypeCmdMySQLStart:
				mu.Lock()
				running = true
				mu.Unlock()
				resPayload, _ := json.Marshal(mysqlRunningStatus(true))
				_ = conn.WriteJSON(protocol.Envelope{
					V: protocol.Version, Type: protocol.TypeResMySQLStart, RequestID: env.RequestID, Payload: resPayload,
				})
			case protocol.TypeCmdMySQLStatus:
				mu.Lock()
				isRunning := running
				mu.Unlock()
				resPayload, _ := json.Marshal(mysqlRunningStatus(isRunning))
				_ = conn.WriteJSON(protocol.Envelope{
					V: protocol.Version, Type: protocol.TypeResMySQLStatus, RequestID: env.RequestID, Payload: resPayload,
				})
			case protocol.TypeCmdMySQLStop:
				mu.Lock()
				running = false
				mu.Unlock()
				_ = conn.WriteJSON(protocol.Envelope{
					V: protocol.Version, Type: protocol.TypeResMySQLStop, RequestID: env.RequestID, Payload: []byte(`{"status":"ok"}`),
				})
			case protocol.TypeCmdMySQLUninstall:
				mu.Lock()
				running = false
				mu.Unlock()
				_ = conn.WriteJSON(protocol.Envelope{
					V: protocol.Version, Type: protocol.TypeResMySQLUninstall, RequestID: env.RequestID, Payload: []byte(`{"status":"ok"}`),
				})
			case protocol.TypeCmdMySQLDatabaseCreate, protocol.TypeCmdMySQLDatabaseDrop,
				protocol.TypeCmdMySQLUserCreate, protocol.TypeCmdMySQLUserDrop, protocol.TypeCmdMySQLUserGrant:
				_ = conn.WriteJSON(protocol.Envelope{
					V: protocol.Version, Type: stringsResType(env.Type), RequestID: env.RequestID, Payload: []byte(`{"status":"ok"}`),
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

	installed, err := svc.InstallMySQL(ctx, "owner-1", view.ID, MySQLInstallInput{
		Engine: "mariadb", Version: "8", Method: "docker",
	})
	if err != nil {
		t.Fatal(err)
	}
	if installed.Status != models.MySQLStatusInstalling && installed.Status != models.MySQLStatusStarting && installed.Status != models.MySQLStatusRunning {
		t.Fatalf("install status: %s", installed.Status)
	}

	deadline := time.Now().Add(2 * time.Second)
	var got *MySQLView
	for time.Now().Before(deadline) {
		got, err = svc.GetMySQL(ctx, "owner-1", view.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == models.MySQLStatusRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.Status != models.MySQLStatusRunning {
		t.Fatalf("expected running, got %+v", got)
	}
	if got.RootPassword == "" || got.HostLocal != "127.0.0.1" || got.Image != "mariadb:11.4" {
		t.Fatalf("connection: %+v", got)
	}

	got, err = svc.CreateMySQLDatabase(ctx, "owner-1", view.ID, "survival")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Databases) != 1 || got.Databases[0].Name != "survival" {
		t.Fatalf("databases: %+v", got.Databases)
	}
	got, err = svc.CreateMySQLUser(ctx, "owner-1", view.ID, MySQLUserInput{
		Username: "plugin",
		Host:     "%",
		Grants:   []MySQLGrantInput{{Database: "survival", Privileges: []string{"SELECT", "INSERT"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Users) != 1 || got.Users[0].Username != "plugin" || got.Users[0].Password == "" {
		t.Fatalf("users: %+v", got.Users)
	}
	if len(got.Users[0].Grants) != 1 || got.Users[0].Grants[0].Database != "survival" {
		t.Fatalf("grants: %+v", got.Users[0].Grants)
	}

	if _, err := svc.CreateMySQLDatabase(ctx, "owner-1", view.ID, "bad-name"); !errors.Is(err, ErrMySQLInvalidIdent) {
		t.Fatalf("expected invalid ident: %v", err)
	}

	stopped, err := svc.StopMySQL(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != models.MySQLStatusInstalled {
		t.Fatalf("stop status: %s", stopped.Status)
	}

	removed, err := svc.UninstallMySQL(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Status != models.MySQLStatusNotInstalled {
		t.Fatalf("uninstall status: %s", removed.Status)
	}
	if len(removed.Databases) != 0 || len(removed.Users) != 0 {
		t.Fatalf("expected empty mysql after uninstall: %+v", removed)
	}
}

func TestMergeMySQLStatusKeepsError(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")
	now := time.Now().UTC()
	if err := svc.db.WithContext(ctx).Create(&models.ServerMySQL{
		ServerID:  view.ID,
		Status:    models.MySQLStatusError,
		LastError: "percona-server-server has no installation candidate",
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(mysqlRunningStatus(false))
	svc.mergeMySQLStatus(ctx, view.ID, payload)
	got, err := svc.GetMySQL(ctx, "owner-1", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.MySQLStatusError {
		t.Fatalf("status: %s", got.Status)
	}
	if got.LastError == "" {
		t.Fatal("expected last_error to remain")
	}
}

func mysqlRunningStatus(running bool) protocol.MySQLStatusResult {
	return protocol.MySQLStatusResult{
		Installed: true,
		Running:   running,
		Engine:    "mariadb",
		Version:   "8.0",
		Method:    "docker",
		BindAddr:  "127.0.0.1",
		Port:      3306,
		Image:     "mariadb:11.4",
	}
}

func stringsResType(cmd string) string {
	switch cmd {
	case protocol.TypeCmdMySQLDatabaseCreate:
		return protocol.TypeResMySQLDatabaseCreate
	case protocol.TypeCmdMySQLDatabaseDrop:
		return protocol.TypeResMySQLDatabaseDrop
	case protocol.TypeCmdMySQLUserCreate:
		return protocol.TypeResMySQLUserCreate
	case protocol.TypeCmdMySQLUserDrop:
		return protocol.TypeResMySQLUserDrop
	case protocol.TypeCmdMySQLUserGrant:
		return protocol.TypeResMySQLUserGrant
	default:
		return cmd
	}
}
