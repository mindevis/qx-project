package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestUpdateInstanceResourceSide(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	owner := Owner{UserID: "user-side"}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Side Test",
		MCVersion:     "1.21",
		Loader:        models.LoaderFabric,
		LoaderVersion: "0.15.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	inst := createRes.Instance

	stored, err := svc.GetInstance(ctx, owner, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.Mods = models.InstanceResourceList{
		{
			Source:       "modrinth",
			ProjectID:    "sodium",
			ProjectName:  "Sodium",
			Filename:     "sodium.jar",
			ResourceType: "mod",
			InstalledAt:  resourceInstalledAt(),
		},
	}
	if err := db.Save(stored).Error; err != nil {
		t.Fatal(err)
	}

	if err := svc.UpdateInstanceResourceSide(ctx, owner, inst.ID, UpdateInstanceResourceSideInput{
		Source:       "modrinth",
		ProjectID:    "sodium",
		ResourceType: "mod",
		SideOverride: "client",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := svc.ListInstanceResources(ctx, owner, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SideOverride != "client" {
		t.Fatalf("side override: %+v", items)
	}
}
