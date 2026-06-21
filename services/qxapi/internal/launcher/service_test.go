package launcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func newLauncherService(t *testing.T) (*Service, *gorm.DB, *auth.TokenService) {
	t.Helper()
	db := testutil.OpenSQLiteDB(t)
	tokens := auth.NewTokenService("test-secret", time.Minute, time.Hour)
	return NewService(db, tokens, "http://localhost:5173"), db, tokens
}

func TestRegisterAndLinkDeviceGuest(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	reg, err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		DeviceID: "device-1",
		OS:       "windows",
	})
	if err != nil || reg.LinkURL == "" {
		t.Fatalf("register: err=%v reg=%+v", err, reg)
	}

	status, err := svc.DeviceStatus(ctx, "device-1")
	if err != nil || status.Status != models.DeviceStatusPendingLink {
		t.Fatalf("status pending: err=%v status=%+v", err, status)
	}

	link, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "device-1"})
	if err != nil || link.GuestToken == "" || link.OwnerType != "guest" {
		t.Fatalf("link guest: err=%v link=%+v", err, link)
	}

	status, err = svc.DeviceStatus(ctx, "device-1")
	if err != nil || status.Status != models.DeviceStatusLinked || status.DeviceToken == nil {
		t.Fatalf("status linked: err=%v status=%+v", err, status)
	}
}

func TestLinkDeviceAsUser(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "device-user"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	link, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "device-user", UserID: "user-1"})
	if err != nil || link.OwnerType != "user" || link.GuestToken != "" {
		t.Fatalf("link user: err=%v link=%+v", err, link)
	}
}

func TestInstancesCRUD(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1", IsGuest: false}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:      "Survival",
		MCVersion: "1.21",
		Loader:    models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	items, err := svc.ListInstances(ctx, owner)
	if err != nil || len(items) != 1 {
		t.Fatalf("list: err=%v len=%d", err, len(items))
	}

	if err := svc.DeleteInstance(ctx, owner, inst.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.DeleteInstance(ctx, owner, inst.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete again: %v", err)
	}
}

func TestGuestVanillaOnly(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{GuestSessionID: "guest-1", IsGuest: true}

	_, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:      "Modded",
		MCVersion: "1.21",
		Loader:    "fabric",
	})
	if !errors.Is(err, ErrGuestLoaderOnly) {
		t.Fatalf("expected guest loader error, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	_, err := svc.RegisterDevice(context.Background(), RegisterDeviceInput{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func TestLinkExpiredAndWrongCode(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()

	reg, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "expired-device"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	past := time.Now().UTC().Add(-time.Hour)
	if err := db.Model(&models.LauncherDevice{}).Where("device_id = ?", "expired-device").Update("link_expires_at", past).Error; err != nil {
		t.Fatalf("expire: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "expired-device"}); !errors.Is(err, ErrLinkExpired) {
		t.Fatalf("expected link expired, got %v", err)
	}
	_ = reg
}

func TestDeviceMe(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "me-dev"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "me-dev", UserID: "user-1"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	info, err := svc.DeviceMe(ctx, "me-dev")
	if err != nil || info.OwnerType != "user" || info.UserID == nil || *info.UserID != "user-1" {
		t.Fatalf("me: err=%v info=%+v", err, info)
	}
}

func TestListInstancesForDevice(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "sync-guest"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "sync-guest"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	device, err := svc.getDevice(ctx, "sync-guest")
	if err != nil || device.GuestSessionID == nil {
		t.Fatalf("device: err=%v", err)
	}
	owner := Owner{GuestSessionID: *device.GuestSessionID, IsGuest: true}

	_, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "A", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	items, err := svc.ListInstancesForDevice(ctx, "sync-guest")
	if err != nil || len(items) != 1 {
		t.Fatalf("list for device: err=%v len=%d", err, len(items))
	}

	_, err = svc.ListInstancesForDevice(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing device: %v", err)
	}
}

func TestUnlinkDevice(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "unlink-dev"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "unlink-dev", UserID: "user-1"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	result, err := svc.UnlinkDevice(ctx, "unlink-dev")
	if err != nil || result.Status != models.DeviceStatusPendingLink {
		t.Fatalf("unlink: err=%v result=%+v", err, result)
	}

	device, err := svc.getDevice(ctx, "unlink-dev")
	if err != nil || device.Status != models.DeviceStatusPendingLink || device.UserID != nil {
		t.Fatalf("after unlink: err=%v device=%+v", err, device)
	}

	_, err = svc.UnlinkDevice(ctx, "unlink-dev")
	if !errors.Is(err, ErrDeviceNotLinked) {
		t.Fatalf("double unlink: %v", err)
	}

	_, err = svc.UnlinkDevice(ctx, "")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("empty id: %v", err)
	}

	_, err = svc.UnlinkDevice(ctx, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestLinkDeviceGuestRelinkAfterUnlink(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "relink-dev"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "relink-dev"}); err != nil {
		t.Fatalf("link guest: %v", err)
	}
	if _, err := svc.UnlinkDevice(ctx, "relink-dev"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "relink-dev"}); err != nil {
		t.Fatalf("re-register: %v", err)
	}
	link, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "relink-dev"})
	if err != nil || link.GuestToken == "" {
		t.Fatalf("relink guest: err=%v link=%+v", err, link)
	}
}

func TestUnlinkDeviceForOwner(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "owner-unlink"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "owner-unlink"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	device, err := svc.getDevice(ctx, "owner-unlink")
	if err != nil || device.GuestSessionID == nil {
		t.Fatalf("device: err=%v", err)
	}
	owner := Owner{GuestSessionID: *device.GuestSessionID, IsGuest: true}

	result, err := svc.UnlinkDeviceForOwner(ctx, owner)
	if err != nil || result.Status != models.DeviceStatusPendingLink {
		t.Fatalf("unlink owner: err=%v result=%+v", err, result)
	}

	_, err = svc.UnlinkDeviceForOwner(ctx, Owner{UserID: "nobody", IsGuest: false})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("no device: %v", err)
	}
}
