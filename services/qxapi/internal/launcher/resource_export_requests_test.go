package launcher

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestFindUploadedInstanceResource(t *testing.T) {
	inst := &models.LauncherInstance{
		Mods: models.InstanceResourceList{
			{
				Source:       "upload",
				ProjectName:  "Custom",
				Filename:     "custom.jar",
				ResourceType: "mod",
			},
			{
				Source:       "modrinth",
				ProjectID:    "sodium",
				ProjectName:  "Sodium",
				Filename:     "sodium.jar",
				ResourceType: "mod",
			},
		},
	}

	if got := FindUploadedInstanceResource(inst, "custom.jar", "mod"); got == nil {
		t.Fatal("expected uploaded mod")
	}
	if got := FindUploadedInstanceResource(inst, "sodium.jar", "mod"); got != nil {
		t.Fatal("catalog mod should not match upload lookup")
	}
	if got := FindUploadedInstanceResource(inst, "missing.jar", "mod"); got != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestExportInstanceResourceDoesNotStoreJarInSQL(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-export"}

	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		DeviceID: "device-export", OS: "windows", Hostname: "test", LauncherVersion: "0.1.0",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "device-export", UserID: owner.UserID}); err != nil {
		t.Fatalf("link: %v", err)
	}
	created, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Pack", MCVersion: "1.21.1", Loader: models.LoaderNeoForge, LoaderVersion: "21.1.248",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	payload := []byte("exported-jar-bytes")
	done := make(chan error, 1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		item, err := svc.FetchPendingResourceExport(ctx, "device-export")
		if err != nil {
			done <- err
			return
		}
		if item == nil {
			done <- fmt.Errorf("expected pending export")
			return
		}
		_, err = svc.UpdateResourceExportRequest(ctx, "device-export", item.ID, UpdateResourceExportRequestInput{
			Status:     models.ResourceExportStatusCompleted,
			ContentB64: base64.StdEncoding.EncodeToString(payload),
		})
		done <- err
	}()

	data, err := svc.ExportInstanceResource(ctx, owner, created.Instance.ID, "Cataclysm.jar", "mod")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if string(data) != string(payload) {
		t.Fatalf("export data = %q", data)
	}
	if err := <-done; err != nil {
		t.Fatalf("launcher side: %v", err)
	}

	var row models.InstanceResourceExportRequest
	if err := db.Where("instance_id = ?", created.Instance.ID).First(&row).Error; err != nil {
		t.Fatalf("load row: %v", err)
	}
	if row.FileSize != int64(len(payload)) {
		t.Fatalf("file_size = %d, want %d", row.FileSize, len(payload))
	}
	assertNoContentB64Column(t, db, "instance_resource_export_requests")
}
