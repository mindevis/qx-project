package launcher

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestResourceInstallFolder(t *testing.T) {
	tests := map[string]string{
		"mod":          "mods",
		"resourcepack": "resourcepacks",
		"shader":       "shaderpacks",
		"datapack":     filepath.Join("saves", "world", "datapacks"),
	}
	for resourceType, want := range tests {
		if got := ResourceInstallFolder(resourceType); got != want {
			t.Fatalf("ResourceInstallFolder(%q) = %q, want %q", resourceType, got, want)
		}
	}
}

func TestNormalizeResourceTypeDatapack(t *testing.T) {
	if got := normalizeResourceType("datapack"); got != "datapack" {
		t.Fatalf("normalizeResourceType(datapack) = %q", got)
	}
}

func TestUpdateModInstallRequestRecordsResource(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	owner := Owner{UserID: "user-mi"}
	const deviceID = "device-mi"

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Forge", MCVersion: "1.20.1", Loader: models.LoaderForge, LoaderVersion: "47.4.0",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	instID := createRes.Instance.ID

	complete := func(reqID, projectID, name, file string) {
		req := models.ModInstallRequest{
			ID:           reqID,
			DeviceID:     deviceID,
			InstanceID:   instID,
			Source:       "curseforge",
			ProjectID:    projectID,
			ProjectName:  name,
			VersionID:    "ver-" + projectID,
			Filename:     file,
			DownloadURL:  "https://example.com/" + file,
			ResourceType: "mod",
			Status:       models.ModInstallStatusDispatched,
		}
		if err := db.Create(&req).Error; err != nil {
			t.Fatalf("create request %s: %v", reqID, err)
		}
		view, err := svc.UpdateModInstallRequest(ctx, deviceID, reqID, UpdateModInstallRequestInput{
			Status: models.ModInstallStatusCompleted,
		})
		if err != nil {
			t.Fatalf("update request %s: %v", reqID, err)
		}
		if view.Status != models.ModInstallStatusCompleted {
			t.Fatalf("request %s status = %q, want completed", reqID, view.Status)
		}
	}

	// EMI Loot depends on EMI: two mods land on the same instance back to back.
	complete("req-emi", "emi", "EMI", "emi.jar")
	complete("req-emiloot", "emiloot", "EMI Loot", "emi_loot.jar")

	items, err := svc.ListInstanceResources(ctx, owner, instID)
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	got := map[string]bool{}
	for _, it := range items {
		got[it.ProjectID] = true
	}
	if !got["emi"] || !got["emiloot"] {
		t.Fatalf("expected both emi and emiloot recorded, got %+v", items)
	}
}

func TestFetchPendingModInstallClaimsOwnerRequestFromAnotherDevice(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	owner := Owner{UserID: "user-multi"}
	uid := owner.UserID
	now := time.Now().UTC()

	// The owner has two linked devices; only "device-active" is polling.
	for _, dev := range []string{"device-old", "device-active"} {
		if err := db.Create(&models.LauncherDevice{
			ID:       "row-" + dev,
			DeviceID: dev,
			UserID:   &uid,
			Status:   models.DeviceStatusLinked,
			LinkedAt: &now,
		}).Error; err != nil {
			t.Fatalf("create device %s: %v", dev, err)
		}
	}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Forge", MCVersion: "1.20.1", Loader: models.LoaderForge, LoaderVersion: "47.4.0",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// The web bound this request to the inactive device.
	req := models.ModInstallRequest{
		ID:           "req-emiloot",
		DeviceID:     "device-old",
		InstanceID:   createRes.Instance.ID,
		Source:       "curseforge",
		ProjectID:    "681783",
		ProjectName:  "EMI Loot",
		VersionID:    "7417249",
		Filename:     "emi_loot.jar",
		DownloadURL:  "https://example.com/emi_loot.jar",
		ResourceType: "mod",
		Status:       models.ModInstallStatusQueued,
		ExpiresAt:    now.Add(time.Hour),
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}

	view, err := svc.FetchPendingModInstall(ctx, "device-active")
	if err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if view == nil {
		t.Fatal("expected the active device to claim the pending request, got nil")
	}
	if view.ID != "req-emiloot" || view.Status != models.ModInstallStatusDispatched {
		t.Fatalf("unexpected view %+v", view)
	}

	var stored models.ModInstallRequest
	if err := db.First(&stored, "id = ?", "req-emiloot").Error; err != nil {
		t.Fatalf("reload request: %v", err)
	}
	if stored.DeviceID != "device-active" {
		t.Fatalf("request device_id = %q, want it claimed by device-active", stored.DeviceID)
	}
}

func TestUpdateModInstallRequestKeepsPendingWhenRecordFails(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	const deviceID = "device-mi"

	// Point the request at a non-existent instance so recording the resource fails.
	req := models.ModInstallRequest{
		ID:           "req-orphan",
		DeviceID:     deviceID,
		InstanceID:   "missing-instance",
		Source:       "curseforge",
		ProjectID:    "emiloot",
		ProjectName:  "EMI Loot",
		VersionID:    "ver-1",
		Filename:     "emi_loot.jar",
		DownloadURL:  "https://example.com/emi_loot.jar",
		ResourceType: "mod",
		Status:       models.ModInstallStatusDispatched,
	}
	if err := db.Create(&req).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}

	if _, err := svc.UpdateModInstallRequest(ctx, deviceID, "req-orphan", UpdateModInstallRequestInput{
		Status: models.ModInstallStatusCompleted,
	}); err == nil {
		t.Fatal("expected error when recording the resource fails")
	}

	var stored models.ModInstallRequest
	if err := db.First(&stored, "id = ?", "req-orphan").Error; err != nil {
		t.Fatalf("reload request: %v", err)
	}
	if stored.Status == models.ModInstallStatusCompleted {
		t.Fatal("request must not be marked completed when the resource record fails")
	}
}
