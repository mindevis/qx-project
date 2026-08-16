package launcher

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

func assertNoContentB64Column(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	type col struct {
		Name string `gorm:"column:name"`
	}
	var cols []col
	if err := db.Raw("PRAGMA table_info(" + table + ")").Scan(&cols).Error; err != nil {
		t.Fatalf("pragma table_info %s: %v", table, err)
	}
	for _, c := range cols {
		if strings.EqualFold(c.Name, "content_b64") {
			t.Fatalf("%s must not have content_b64", table)
		}
	}
}

func TestCreateInstanceResourceUploadStoresObjectNotSQL(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-blob"}

	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		DeviceID: "device-blob", OS: "windows", Hostname: "test", LauncherVersion: "0.1.0",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "device-blob", UserID: owner.UserID}); err != nil {
		t.Fatalf("link: %v", err)
	}
	created, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Pack", MCVersion: "1.21.1", Loader: models.LoaderNeoForge, LoaderVersion: "21.1.248",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	payload := []byte("fake-jar-bytes")
	done := make(chan error, 1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		item, err := svc.FetchPendingResourceUpload(ctx, "device-blob")
		if err != nil {
			done <- err
			return
		}
		if item == nil {
			done <- fmt.Errorf("expected pending upload")
			return
		}
		if item.Filename != "Cataclysm.jar" {
			done <- fmt.Errorf("filename = %q", item.Filename)
			return
		}
		rc, name, size, err := svc.OpenResourceUpload(ctx, "device-blob", item.ID)
		if err != nil {
			done <- err
			return
		}
		defer rc.Close()
		got, err := io.ReadAll(rc)
		if err != nil {
			done <- err
			return
		}
		if name != "Cataclysm.jar" || size != int64(len(payload)) || string(got) != string(payload) {
			done <- fmt.Errorf("blob name=%q size=%d data=%q", name, size, got)
			return
		}
		_, err = svc.UpdateResourceUploadRequest(ctx, "device-blob", item.ID, UpdateResourceUploadRequestInput{
			Status: models.ResourceUploadStatusCompleted,
		})
		done <- err
	}()

	view, err := svc.CreateInstanceResourceUpload(ctx, owner, created.Instance.ID, "Cataclysm.jar", "mod", payload)
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	if view.Filename != "Cataclysm.jar" {
		t.Fatalf("view = %+v", view)
	}
	if err := <-done; err != nil {
		t.Fatalf("launcher side: %v", err)
	}

	var row models.InstanceResourceUploadRequest
	if err := db.Where("id = ?", view.ID).First(&row).Error; err != nil {
		t.Fatalf("load row: %v", err)
	}
	if row.ObjectKey == "" {
		t.Fatal("expected object key")
	}
	assertNoContentB64Column(t, db, "instance_resource_upload_requests")
}
