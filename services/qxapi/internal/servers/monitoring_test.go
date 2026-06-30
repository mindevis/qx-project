package servers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

// Collation mismatch (utf8mb4_unicode_ci vs utf8mb4_0900_ai_ci) only surfaces on MySQL
// when game_servers was AutoMigrate'd without matching docs/schema.sql. SQLite tests
// below do not catch it; see docs/migrations/2026-06-30_game_servers_collation.sql.

func TestMonitoringJoinClausesUseUnicodeCollation(t *testing.T) {
	require.Contains(t, monitoringJoinServersMySQL, "COLLATE "+mysqlUnicodeCI)
	require.Contains(t, monitoringJoinServersMySQL, "game_servers.server_id")
	require.Equal(t, monitoringJoinServersPlain, monitoringJoinServers(testutil.OpenSQLiteDB(t)))
	require.True(t, strings.Contains(monitoringJoinUsersSQL, "servers.owner_id"))
}

func TestListMonitoringServers_PremiumFirst(t *testing.T) {
	svc, _, _ := newServersService(t)
	ctx := context.Background()
	db := testutil.OpenSQLiteDB(t)
	svc = NewService(db, nil, nil, nil, NoopDeployer{})

	freeOwnerID := "owner-free"
	premiumOwnerID := "owner-premium"
	now := time.Now().UTC()
	require.NoError(t, db.Create(&models.User{
		ID: freeOwnerID, Email: "free@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.User{
		ID: premiumOwnerID, Email: "premium@example.com", PasswordHash: "x", Tier: "premium", CreatedAt: now, UpdatedAt: now,
	}).Error)

	freeVPSID := "dedicated server-free"
	premiumVPSID := "dedicated server-premium"
	require.NoError(t, db.Create(&models.Server{
		ID: freeVPSID, OwnerID: freeOwnerID, Name: "Free Dedicated", Status: models.ServerStatusPending, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: premiumVPSID, OwnerID: premiumOwnerID, Name: "Premium Dedicated", Status: models.ServerStatusPending, CreatedAt: now, UpdatedAt: now,
	}).Error)

	freeAddr := "1.2.3.4"
	premiumAddr := "5.6.7.8"
	require.NoError(t, db.Create(&models.GameServer{
		ID: "11111111-1111-1111-1111-111111111111", ServerID: freeVPSID, Name: "Free Server",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.20.1", Address: &freeAddr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: true, LikesCount: 100, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServer{
		ID: "22222222-2222-2222-2222-222222222222", ServerID: premiumVPSID, Name: "Premium Server",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.20.1", Address: &premiumAddr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: true, LikesCount: 1,
		CreatedAt: now.Add(time.Minute), UpdatedAt: now,
	}).Error)

	items, err := svc.ListMonitoringServers(ctx, ListMonitoringInput{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.True(t, items[0].IsPremium)
	require.Equal(t, "22222222-2222-2222-2222-222222222222", items[0].ID)
}

func TestListMonitoringServers_FiltersHidden(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	ctx := context.Background()
	ownerID := "owner-1"
	now := time.Now().UTC()
	vpsID := "dedicated server-1"
	addr := "9.9.9.9"

	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "owner@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: vpsID, OwnerID: ownerID, Name: "Dedicated", Status: models.ServerStatusPending, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServer{
		ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", ServerID: vpsID, Name: "Visible",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.21", Address: &addr, Port: 25565,
		Status: models.GameServerStatusStopped, ShowInMonitoring: true, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServer{
		ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", ServerID: vpsID, Name: "Hidden",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.21", Address: &addr, Port: 25566,
		Status: models.GameServerStatusStopped, ShowInMonitoring: false, CreatedAt: now, UpdatedAt: now,
	}).Error)

	items, err := svc.ListMonitoringServers(ctx, ListMonitoringInput{MCVersion: "1.21"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", items[0].ID)
}
