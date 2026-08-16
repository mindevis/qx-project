package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestAssertInstanceContentMutable_ManagedByField(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	userID := "managed-user"
	gsID := "gs-managed-1"
	instID := "inst-managed-1"
	require.NoError(t, db.Create(&models.User{
		ID: userID, Email: "managed@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &userID, Name: "Server Client", MCVersion: "1.21", Loader: models.LoaderForge,
		ManagedByGameServerID: &gsID, CreatedAt: now, UpdatedAt: now,
	}).Error)

	require.ErrorIs(t, svc.AssertInstanceContentMutable(ctx, instID), ErrInstanceManaged)
}

func TestAssertInstanceContentMutable_BoundInstance(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	userID := "bound-user"
	instID := "inst-bound-1"
	require.NoError(t, db.Create(&models.User{
		ID: userID, Email: "bound@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &userID, Name: "Legacy Bound", MCVersion: "1.21", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.GameServerInstanceBinding{
		ID: "bind-1", UserID: userID, GameServerID: "gs-legacy", LauncherInstanceID: instID,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	require.ErrorIs(t, svc.AssertInstanceContentMutable(ctx, instID), ErrInstanceManaged)
}

func TestAssertInstanceContentMutable_PersonalInstance(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	userID := "personal-user"
	instID := "inst-personal-1"
	require.NoError(t, db.Create(&models.User{
		ID: userID, Email: "personal@example.com", PasswordHash: "x", Tier: "free", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&models.LauncherInstance{
		ID: instID, UserID: &userID, Name: "Personal", MCVersion: "1.21", Loader: models.LoaderVanilla,
		CreatedAt: now, UpdatedAt: now,
	}).Error)

	require.NoError(t, svc.AssertInstanceContentMutable(ctx, instID))
}
