package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestPrepareRequestLifecycle(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()

	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "prep-svc-dev"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "prep-svc-dev", UserID: "user-prep-svc"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	createRes, err := svc.CreateInstance(ctx, Owner{UserID: "user-prep-svc"}, CreateInstanceInput{
		Name:      "Test",
		MCVersion: "1.21",
		Loader:    models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if createRes.PrepareRequestID == nil {
		t.Fatal("expected prepare request")
	}

	pending, err := svc.FetchPendingPrepare(ctx, "prep-svc-dev")
	if err != nil || pending == nil {
		t.Fatalf("fetch pending: err=%v pending=%+v", err, pending)
	}
	if pending.Status != models.PrepareStatusPreparing {
		t.Fatalf("expected preparing after fetch, got %s", pending.Status)
	}
	if pending.Instance == nil || pending.Instance.ID != createRes.Instance.ID {
		t.Fatalf("expected instance metadata, got %+v", pending.Instance)
	}

	view, err := svc.UpdatePrepareRequest(ctx, "prep-svc-dev", pending.ID, UpdatePrepareRequestInput{
		Status: models.PrepareStatusDownloading,
	})
	if err != nil || view.Status != models.PrepareStatusDownloading {
		t.Fatalf("update downloading: err=%v view=%+v", err, view)
	}

	view, err = svc.UpdatePrepareRequest(ctx, "prep-svc-dev", pending.ID, UpdatePrepareRequestInput{
		Status: models.PrepareStatusCompleted,
	})
	if err != nil || view.Status != models.PrepareStatusCompleted {
		t.Fatalf("update completed: err=%v view=%+v", err, view)
	}

	ownerView, err := svc.GetPrepareRequest(ctx, Owner{UserID: "user-prep-svc"}, pending.ID)
	if err != nil || ownerView.Status != models.PrepareStatusCompleted {
		t.Fatalf("get for owner: err=%v view=%+v", err, ownerView)
	}
}
