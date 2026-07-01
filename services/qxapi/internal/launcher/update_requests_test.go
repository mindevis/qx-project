package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestReleaseAndUpdateRequests(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("secret", time.Minute, time.Hour)
	svc := NewService(db, tokens, "https://mc.qx-dev.ru")
	svc.SetRelease("1.0.0", "/downloads/qx-launcher.exe")

	release := svc.GetRelease()
	if release.Version != "1.0.0" || release.DownloadURL != "/downloads/qx-launcher.exe" {
		t.Fatalf("release: %#v", release)
	}

	ctx := context.Background()
	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "svc-upd"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "svc-upd", UserID: "user-1"}); err != nil {
		t.Fatalf("link: %v", err)
	}
	if _, err := svc.RequestLauncherUpdate(ctx, "user-1"); err != nil {
		t.Fatalf("request: %v", err)
	}
	item, err := svc.FetchPendingUpdate(ctx, "svc-upd")
	if err != nil || item == nil {
		t.Fatalf("pending: item=%v err=%v", item, err)
	}
	if err := svc.CompleteLauncherUpdate(ctx, "svc-upd", CompleteUpdateInput{
		Status:          "completed",
		LauncherVersion: "1.0.0",
	}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	pending, err := svc.FetchPendingUpdate(ctx, "svc-upd")
	if err != nil || pending != nil {
		t.Fatalf("expected no pending update, got %v err=%v", pending, err)
	}
}
