package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestDeleteInstanceResource(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	owner := Owner{UserID: "user-res"}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Forge", MCVersion: "1.21", Loader: models.LoaderForge, LoaderVersion: "47.0.0",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	inst := createRes.Instance

	var stored models.LauncherInstance
	if err := db.First(&stored, "id = ?", inst.ID).Error; err != nil {
		t.Fatalf("load instance: %v", err)
	}
	stored.Mods = models.InstanceResourceList{
		{Source: "modrinth", ProjectID: "sodium", ProjectName: "Sodium", Filename: "sodium.jar", ResourceType: "mod"},
		{Source: "modrinth", ProjectID: "lithium", ProjectName: "Lithium", Filename: "lithium.jar", ResourceType: "mod"},
	}
	if err := db.Save(&stored).Error; err != nil {
		t.Fatalf("save mods: %v", err)
	}

	if err := svc.DeleteInstanceResource(ctx, owner, inst.ID, DeleteInstanceResourceInput{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium.jar", ResourceType: "mod",
	}); err != nil {
		t.Fatalf("delete resource: %v", err)
	}

	items, err := svc.ListInstanceResources(ctx, owner, inst.ID)
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(items) != 1 || items[0].ProjectID != "lithium" {
		t.Fatalf("expected lithium only, got %+v", items)
	}
}

func TestDeleteInstanceResourceNotFound(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	owner := Owner{UserID: "user-res"}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Forge", MCVersion: "1.21", Loader: models.LoaderForge, LoaderVersion: "47.0.0",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	inst := createRes.Instance

	err = svc.DeleteInstanceResource(ctx, owner, inst.ID, DeleteInstanceResourceInput{
		Source: "modrinth", ProjectID: "missing", ResourceType: "mod",
	})
	if err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
