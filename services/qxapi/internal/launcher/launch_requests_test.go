package launcher

import (
	"context"
	"errors"
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestProfilesCRUD(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1", IsGuest: false}

	profile, err := svc.CreateProfile(ctx, owner, CreateProfileInput{Username: "Steve"})
	if err != nil || profile.Username != "Steve" {
		t.Fatalf("create profile: err=%v p=%+v", err, profile)
	}

	items, err := svc.ListProfiles(ctx, owner)
	if err != nil || len(items) != 1 {
		t.Fatalf("list profiles: err=%v len=%d", err, len(items))
	}

	if err := svc.DeleteProfile(ctx, owner, profile.ID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
}

func TestProfileValidation(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	_, err := svc.CreateProfile(context.Background(), Owner{UserID: "u"}, CreateProfileInput{Username: "x"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation, got %v", err)
	}
}

func TestLaunchRequestFlow(t *testing.T) {
	svc, _, tokens := newLauncherService(t)
	svc.withStubManifest(t, nil)
	ctx := context.Background()

	reg, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-launch"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_ = reg

	link, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-launch"})
	if err != nil {
		t.Fatalf("link: %v", err)
	}
	_ = link
	device, err := svc.getDevice(ctx, "dev-launch")
	if err != nil || device.GuestSessionID == nil {
		t.Fatalf("device: err=%v guest=%v", err, device)
	}
	owner := Owner{GuestSessionID: *device.GuestSessionID, IsGuest: true}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("instance: %v", err)
	}

	profile, err := svc.CreateProfile(ctx, owner, CreateProfileInput{Username: "GuestPlayer"})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}

	created, err := svc.CreateLaunchRequest(ctx, owner, CreateLaunchRequestInput{
		InstanceID:       inst.ID,
		OfflineProfileID: profile.ID,
		DeviceID:         "dev-launch",
	})
	if err != nil || created.Status != models.LaunchStatusQueued {
		t.Fatalf("create launch: err=%v view=%+v", err, created)
	}

	pending, err := svc.FetchPendingLaunch(ctx, "dev-launch")
	if err != nil || pending == nil || pending.Status != models.LaunchStatusDispatched {
		t.Fatalf("pending: err=%v view=%+v", err, pending)
	}

	updated, err := svc.UpdateLaunchRequest(ctx, "dev-launch", created.ID, UpdateLaunchRequestInput{
		Status: models.LaunchStatusRunning,
		PID:    intPtr(1234),
	})
	if err != nil || updated.Status != models.LaunchStatusRunning {
		t.Fatalf("update: err=%v view=%+v", err, updated)
	}

	got, err := svc.GetLaunchRequest(ctx, owner, created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("get: err=%v view=%+v", err, got)
	}
	_ = tokens
}

func intPtr(v int) *int { return &v }

func TestFindLinkedDevice(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.FindLinkedDevice(ctx, Owner{UserID: "user-1"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	_, _ = svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-find"})
	_, _ = svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-find", UserID: "user-1"})

	id, err := svc.FindLinkedDevice(ctx, Owner{UserID: "user-1"})
	if err != nil || id != "dev-find" {
		t.Fatalf("find linked: err=%v id=%q", err, id)
	}

	info, err := svc.UserLinkedDevice(ctx, "user-1")
	if err != nil || info.DeviceID != "dev-find" || info.OwnerType != "user" {
		t.Fatalf("user linked device: err=%v info=%+v", err, info)
	}
}

