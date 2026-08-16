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

func TestEnsureInstancePrepared_RequeuesFailedAndSkipsCompleted(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "http://localhost:5173")
	ctx := context.Background()
	owner := Owner{UserID: "user-prep-retry"}

	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "prep-retry-dev"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "prep-retry-dev", UserID: owner.UserID}); err != nil {
		t.Fatalf("link: %v", err)
	}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Retry", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if createRes.PrepareRequestID == nil {
		t.Fatal("expected initial prepare request")
	}

	if _, err := svc.UpdatePrepareRequest(ctx, "prep-retry-dev", *createRes.PrepareRequestID, UpdatePrepareRequestInput{
		Status:          models.PrepareStatusFailed,
		ErrorCode:       strPtr("LIBRARIES_FAILED"),
		ProgressMessage: "libraries",
	}); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	retried, err := svc.EnsureInstancePrepared(ctx, owner, createRes.Instance.ID)
	if err != nil {
		t.Fatalf("ensure after fail: %v", err)
	}
	if retried == nil || *retried == *createRes.PrepareRequestID {
		t.Fatalf("expected a new prepare request, got %+v", retried)
	}

	if _, err := svc.UpdatePrepareRequest(ctx, "prep-retry-dev", *retried, UpdatePrepareRequestInput{
		Status: models.PrepareStatusCompleted,
	}); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	again, err := svc.EnsureInstancePrepared(ctx, owner, createRes.Instance.ID)
	if err != nil {
		t.Fatalf("ensure after complete: %v", err)
	}
	if again != nil {
		t.Fatalf("expected nil after completed prepare, got %s", *again)
	}
}

func strPtr(v string) *string { return &v }
