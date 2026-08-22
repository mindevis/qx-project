package servers

import (
	"context"
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestGameServerNetworkCRUD(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	view := createTestServer(t, svc, "owner-1")

	created, err := svc.CreateGameServerNetwork(ctx, "owner-1", view.ID, "Mini-games")
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	if created.Name != "Mini-games" || created.ID == "" {
		t.Fatalf("created: %+v", created)
	}

	velocityID := "gs-velocity"
	lobbyID := "gs-lobby"
	survivalID := "gs-survival"
	for _, row := range []models.GameServer{
		{
			ID: velocityID, ServerID: view.ID, Name: "Velocity", ServerType: "velocity",
			MCVersion: "3.4.0-SNAPSHOT", Port: 25565, Status: models.GameServerStatusStopped,
			WorkDir: "/opt/qxsystem/server/instances/gs-velocity", JarPath: "/opt/qxsystem/server/instances/gs-velocity/server.jar",
		},
		{
			ID: lobbyID, ServerID: view.ID, Name: "Lobby", ServerType: "paper",
			MCVersion: "1.21", Port: 25566, Status: models.GameServerStatusStopped,
			WorkDir: "/opt/qxsystem/server/instances/gs-lobby", JarPath: "/opt/qxsystem/server/instances/gs-lobby/server.jar",
		},
		{
			ID: survivalID, ServerID: view.ID, Name: "Survival", ServerType: "paper",
			MCVersion: "1.21", Port: 25567, Status: models.GameServerStatusStopped,
			WorkDir: "/opt/qxsystem/server/instances/gs-survival", JarPath: "/opt/qxsystem/server/instances/gs-survival/server.jar",
		},
	} {
		item := row
		if err := svc.db.WithContext(ctx).Create(&item).Error; err != nil {
			t.Fatalf("create game server: %v", err)
		}
	}

	updated, err := svc.UpdateGameServerNetwork(ctx, "owner-1", view.ID, created.ID, "Network", []GameServerNetworkMemberInput{
		{GameServerID: velocityID, Role: models.GameServerNetworkRoleProxy, Alias: "proxy"},
		{GameServerID: lobbyID, Role: models.GameServerNetworkRoleLobby, Alias: "lobby"},
		{GameServerID: survivalID, Role: models.GameServerNetworkRoleBackend, Alias: "survival"},
	}, false)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Network" || len(updated.Members) != 3 {
		t.Fatalf("updated: %+v", updated)
	}
	if updated.Applied {
		t.Fatal("apply was disabled")
	}

	items, err := svc.ListGameServerNetworks(ctx, "owner-1", view.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list: err=%v len=%d", err, len(items))
	}
	if items[0].Members[0].Name != "Velocity" || items[0].Members[1].Alias != "lobby" {
		t.Fatalf("members: %+v", items[0].Members)
	}

	if _, err := svc.UpdateGameServerNetwork(ctx, "owner-1", view.ID, created.ID, "Network", []GameServerNetworkMemberInput{
		{GameServerID: lobbyID, Role: models.GameServerNetworkRoleProxy, Alias: "bad"},
	}, false); err == nil {
		t.Fatal("expected validation: paper cannot be proxy")
	}

	if err := svc.DeleteGameServerNetwork(ctx, "owner-1", view.ID, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	items, err = svc.ListGameServerNetworks(ctx, "owner-1", view.ID)
	if err != nil || len(items) != 0 {
		t.Fatalf("list after delete: err=%v len=%d", err, len(items))
	}
}

func TestNetworkBackendAddress(t *testing.T) {
	got := networkBackendAddress(GameServerNetworkMemberView{Address: "", Port: 25566})
	if got != "127.0.0.1:25566" {
		t.Fatalf("got %q", got)
	}
}
