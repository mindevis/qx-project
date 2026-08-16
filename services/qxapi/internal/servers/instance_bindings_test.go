package servers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func seedMonitoringGameServer(t *testing.T, db *gorm.DB, gameServerID, ownerID, vpsID string) {
	t.Helper()
	now := time.Now().UTC()
	addr := "play.example.com"
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: ownerID + "@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: vpsID, OwnerID: ownerID, Name: "VPS", Status: models.ServerStatusOnline, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServer{
		ID: gameServerID, ServerID: vpsID, Name: "Public Server",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.21", Address: &addr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: true, CreatedAt: now, UpdatedAt: now,
	}).Error)
}

func TestInstanceBinding_SetListClear(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	ctx := context.Background()

	ownerID := "binding-user"
	gameServerID := "gs-binding-1"
	vpsID := "vps-binding-1"
	seedMonitoringGameServer(t, db, gameServerID, "server-owner", vpsID)

	now := time.Now().UTC()
	instID := "inst-binding-1"
	gsID := gameServerID
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "binding-user@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &ownerID, Name: "My Forge", MCVersion: "1.21", Loader: models.LoaderForge,
		ManagedByGameServerID: &gsID,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	items, err := svc.ListInstanceBindings(ctx, ownerID)
	require.NoError(t, err)
	require.Empty(t, items)

	view, err := svc.SetInstanceBinding(ctx, ownerID, gameServerID, instID)
	require.NoError(t, err)
	require.Equal(t, gameServerID, view.GameServerID)
	require.Equal(t, instID, view.InstanceID)
	require.Equal(t, "My Forge", view.InstanceName)
	require.True(t, view.Locked)

	items, err = svc.ListInstanceBindings(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, instID, items[0].InstanceID)

	again, err := svc.SetInstanceBinding(ctx, ownerID, gameServerID, instID)
	require.NoError(t, err)
	require.Equal(t, instID, again.InstanceID)

	inst2ID := "inst-binding-2"
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: inst2ID, UserID: &ownerID, Name: "Other", MCVersion: "1.20.1", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	_, err = svc.SetInstanceBinding(ctx, ownerID, gameServerID, inst2ID)
	require.ErrorIs(t, err, ErrBindingLocked)

	require.ErrorIs(t, svc.ClearInstanceBinding(ctx, ownerID, gameServerID), ErrBindingLocked)
	items, err = svc.ListInstanceBindings(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, instID, items[0].InstanceID)
}

func TestInstanceBinding_RejectsForeignInstance(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	ctx := context.Background()

	gameServerID := "gs-binding-2"
	seedMonitoringGameServer(t, db, gameServerID, "server-owner-2", "vps-binding-2")

	now := time.Now().UTC()
	otherUser := "other-user"
	instID := "inst-other"
	require.NoError(t, db.Create(&models.User{
		ID: otherUser, Email: "other@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &otherUser, Name: "Foreign", MCVersion: "1.21", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	_, err := svc.SetInstanceBinding(ctx, "binding-user-2", gameServerID, instID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestInstanceBinding_OwnServerWithoutMonitoring(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	ctx := context.Background()

	ownerID := "owner-hidden"
	now := time.Now().UTC()
	vpsID := "vps-hidden"
	addr := "hidden.example.com"
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "hidden@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: vpsID, OwnerID: ownerID, Name: "VPS", Status: models.ServerStatusOnline, CreatedAt: now, UpdatedAt: now,
	}).Error)
	gameServerID := "gs-hidden"
	require.NoError(t, db.Create(&models.GameServer{
		ID: gameServerID, ServerID: vpsID, Name: "Hidden",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.21", Address: &addr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: false, CreatedAt: now, UpdatedAt: now,
	}).Error)
	instID := "inst-hidden"
	gsID := gameServerID
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &ownerID, Name: "Mine", MCVersion: "1.21", Loader: models.LoaderVanilla,
		ManagedByGameServerID: &gsID,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	view, err := svc.SetInstanceBinding(ctx, ownerID, gameServerID, instID)
	require.NoError(t, err)
	require.Equal(t, gameServerID, view.GameServerID)

	items, err := svc.ListBindableServers(ctx, ownerID, ListMonitoringInput{MCVersion: "1.21", Loader: "vanilla"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, gameServerID, items[0].ID)
}

func TestInstanceBinding_RejectsForeignHiddenServer(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	ctx := context.Background()

	ownerID := "owner-hidden"
	now := time.Now().UTC()
	vpsID := "vps-hidden"
	addr := "hidden.example.com"
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "hidden@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.Server{
		ID: vpsID, OwnerID: ownerID, Name: "VPS", Status: models.ServerStatusOnline, CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServer{
		ID: "gs-hidden", ServerID: vpsID, Name: "Hidden",
		ServerType: models.ServerTypeVanilla, MCVersion: "1.21", Address: &addr, Port: 25565,
		Status: models.GameServerStatusRunning, ShowInMonitoring: false, CreatedAt: now, UpdatedAt: now,
	}).Error)
	bindingUser := "binding-user-3"
	require.NoError(t, db.Create(&models.User{
		ID: bindingUser, Email: "binding-user-3@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	instID := "inst-hidden"
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &bindingUser, Name: "Mine", MCVersion: "1.21", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	_, err := svc.SetInstanceBinding(ctx, bindingUser, "gs-hidden", instID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestInstanceBinding_RejectsUnmanagedInstance(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	ctx := context.Background()

	ownerID := "binding-user-unmanaged"
	gameServerID := "gs-binding-unmanaged"
	seedMonitoringGameServer(t, db, gameServerID, "server-owner-unmanaged", "vps-binding-unmanaged")

	now := time.Now().UTC()
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "unmanaged@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	instID := "inst-personal"
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &ownerID, Name: "Personal", MCVersion: "1.21", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	_, err := svc.SetInstanceBinding(ctx, ownerID, gameServerID, instID)
	require.ErrorIs(t, err, ErrBindingLocked)
}

func TestEnsureConnectBinding_CreatesAndReusesManagedInstance(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	svc := NewService(db, nil, nil, nil, NoopDeployer{})
	launcherSvc := launcher.NewService(db, nil, "http://localhost")
	ctx := context.Background()

	ownerID := "connect-user"
	gameServerID := "gs-connect-1"
	loaderVer := "47.0.0"
	seedMonitoringGameServer(t, db, gameServerID, "connect-owner", "vps-connect-1")
	require.NoError(t, db.Model(&models.GameServer{}).Where("id = ?", gameServerID).Updates(map[string]any{
		"server_type":    models.LoaderForge,
		"loader_version": loaderVer,
	}).Error)

	now := time.Now().UTC()
	require.NoError(t, db.Create(&models.User{
		ID: ownerID, Email: "connect-user@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)

	view, err := svc.EnsureConnectBinding(ctx, ownerID, gameServerID, launcherSvc)
	require.NoError(t, err)
	require.Equal(t, gameServerID, view.GameServerID)
	require.NotEmpty(t, view.InstanceID)
	require.True(t, view.Locked)
	require.Equal(t, "Public Server", view.InstanceName)

	var inst models.LauncherInstance
	require.NoError(t, db.Where("id = ?", view.InstanceID).First(&inst).Error)
	require.NotNil(t, inst.ManagedByGameServerID)
	require.Equal(t, gameServerID, *inst.ManagedByGameServerID)

	again, err := svc.EnsureConnectBinding(ctx, ownerID, gameServerID, launcherSvc)
	require.NoError(t, err)
	require.Equal(t, view.InstanceID, again.InstanceID)

	count := int64(0)
	require.NoError(t, db.Model(&models.LauncherInstance{}).Where("user_id = ?", ownerID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

