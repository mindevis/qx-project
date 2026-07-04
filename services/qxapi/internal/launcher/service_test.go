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

func TestRegisterAndLinkDevice(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	reg, err := svc.RegisterDevice(ctx, RegisterDeviceInput{
		DeviceID:        "device-1",
		OS:              "windows",
		Hostname:        "DESKTOP-TEST",
		LauncherVersion: "0.1.0",
	})
	if err != nil || reg.LinkURL == "" {
		t.Fatalf("register: err=%v reg=%+v", err, reg)
	}

	status, err := svc.DeviceStatus(ctx, "device-1")
	if err != nil || status.Status != models.DeviceStatusPendingLink {
		t.Fatalf("status pending: err=%v status=%+v", err, status)
	}
	if status.DeviceID != "device-1" || status.Hostname == nil || *status.Hostname != "DESKTOP-TEST" {
		t.Fatalf("status metadata: %+v", status)
	}
	if status.OS == nil || *status.OS != "windows" || status.LauncherVersion == nil || *status.LauncherVersion != "0.1.0" {
		t.Fatalf("status os/version: %+v", status)
	}

	link, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "device-1", UserID: "user-1"})
	if err != nil || link.OwnerType != "user" {
		t.Fatalf("link: err=%v link=%+v", err, link)
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
	if err != nil || link.OwnerType != "user" {
		t.Fatalf("link user: err=%v link=%+v", err, link)
	}
}

func TestInstancesCRUD(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:      "Survival",
		MCVersion: "1.21",
		Loader:    models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	inst := createRes.Instance

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

func TestCreateInstanceModLoader(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	createRes, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Modded",
		MCVersion:     "1.20.1",
		Loader:        models.LoaderFabric,
		LoaderVersion: "0.16.14",
	})
	if err != nil {
		t.Fatalf("create fabric: %v", err)
	}
	inst := createRes.Instance
	if inst.LoaderVersion == nil || *inst.LoaderVersion != "0.16.14" {
		t.Fatalf("loader version: %+v", inst.LoaderVersion)
	}
	if inst.MinMemoryMB == nil || *inst.MinMemoryMB != defaultInstanceMemoryMB {
		t.Fatalf("min memory: %+v", inst.MinMemoryMB)
	}
	if inst.MaxMemoryMB == nil || *inst.MaxMemoryMB != defaultInstanceMemoryMB {
		t.Fatalf("max memory: %+v", inst.MaxMemoryMB)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:      "Bad",
		MCVersion: "1.20.1",
		Loader:    models.LoaderForge,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation without loader version, got %v", err)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Bad",
		MCVersion:     "1.20.1",
		Loader:        "paper",
		LoaderVersion: "1",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for unsupported loader, got %v", err)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Bad NeoForge",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderNeoForge,
		LoaderVersion: "47.4.20",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for forge version on neoforge, got %v", err)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Bad Forge",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderForge,
		LoaderVersion: "21.1.234",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for neoforge version on forge, got %v", err)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "NeoForge",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderNeoForge,
		LoaderVersion: "21.1.234",
	})
	if err != nil {
		t.Fatalf("create neoforge: %v", err)
	}
	inst = createRes.Instance
	if inst.LoaderVersion == nil || *inst.LoaderVersion != "21.1.234" {
		t.Fatalf("loader version: %+v", inst.LoaderVersion)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Bad Fabric",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderFabric,
		LoaderVersion: "47.4.20",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation for forge version on fabric, got %v", err)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Fabric",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderFabric,
		LoaderVersion: "0.19.3",
	})
	if err != nil {
		t.Fatalf("create fabric: %v", err)
	}
	inst = createRes.Instance
	if inst.LoaderVersion == nil || *inst.LoaderVersion != "0.19.3" {
		t.Fatalf("loader version: %+v", inst.LoaderVersion)
	}

	createRes, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Quilt",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderQuilt,
		LoaderVersion: "0.28.1",
	})
	if err != nil {
		t.Fatalf("create quilt: %v", err)
	}
	inst = createRes.Instance
	if inst.LoaderVersion == nil || *inst.LoaderVersion != "0.28.1" {
		t.Fatalf("loader version: %+v", inst.LoaderVersion)
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
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "expired-device", UserID: "user-expired"}); !errors.Is(err, ErrLinkExpired) {
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

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "sync-user"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "sync-user", UserID: "user-sync"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	owner := Owner{UserID: "user-sync"}

	if _, err = svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "A", MCVersion: "1.21", Loader: models.LoaderVanilla,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	items, err := svc.ListInstancesForDevice(ctx, "sync-user")
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

func TestUnlinkDeviceForOwner(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "owner-unlink"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "owner-unlink", UserID: "user-owner"}); err != nil {
		t.Fatalf("link: %v", err)
	}

	owner := Owner{UserID: "user-owner"}

	result, err := svc.UnlinkDeviceForOwner(ctx, owner)
	if err != nil || result.Status != models.DeviceStatusPendingLink {
		t.Fatalf("unlink owner: err=%v result=%+v", err, result)
	}

	_, err = svc.UnlinkDeviceForOwner(ctx, Owner{UserID: "nobody"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("no device: %v", err)
	}
}
