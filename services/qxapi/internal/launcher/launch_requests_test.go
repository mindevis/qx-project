package launcher

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/mojang"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type stubMojangLauncher struct {
	linked       bool
	sessionErr   error
	session      *mojang.SessionView
	sessionCalls int
}

func (s *stubMojangLauncher) GetStatus(ctx context.Context, userID string) (*mojang.LinkStatus, error) {
	return &mojang.LinkStatus{Linked: s.linked}, nil
}

func (s *stubMojangLauncher) SessionForLaunch(ctx context.Context, userID string) (*mojang.SessionView, error) {
	s.sessionCalls++
	if s.sessionErr != nil {
		return nil, s.sessionErr
	}
	return s.session, nil
}

func TestProfilesCRUD(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

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
	ctx := context.Background()

	reg, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-launch"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_ = reg

	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-launch", UserID: "user-launch"}); err != nil {
		t.Fatalf("link: %v", err)
	}
	owner := Owner{UserID: "user-launch"}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("instance: %v", err)
	}

	profile, err := svc.CreateProfile(ctx, owner, CreateProfileInput{Username: "LaunchPlayer"})
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
	if pending.Instance == nil || pending.Instance.MCVersion != "1.21" {
		t.Fatalf("expected instance metadata in pending view, got %+v", pending.Instance)
	}

	updated, err := svc.UpdateLaunchRequest(ctx, "dev-launch", created.ID, UpdateLaunchRequestInput{
		Status: models.LaunchStatusPreparing,
	})
	if err != nil || updated.Status != models.LaunchStatusPreparing {
		t.Fatalf("update preparing: err=%v view=%+v", err, updated)
	}

	updated, err = svc.UpdateLaunchRequest(ctx, "dev-launch", created.ID, UpdateLaunchRequestInput{
		Status: models.LaunchStatusDownloading,
	})
	if err != nil || updated.Status != models.LaunchStatusDownloading {
		t.Fatalf("update downloading: err=%v view=%+v", err, updated)
	}

	updated, err = svc.UpdateLaunchRequest(ctx, "dev-launch", created.ID, UpdateLaunchRequestInput{
		Status: models.LaunchStatusLaunching,
	})
	if err != nil || updated.Status != models.LaunchStatusLaunching {
		t.Fatalf("update launching: err=%v view=%+v", err, updated)
	}

	updated, err = svc.UpdateLaunchRequest(ctx, "dev-launch", created.ID, UpdateLaunchRequestInput{
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

func TestFetchPendingLaunchEmpty(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	pending, err := svc.FetchPendingLaunch(ctx, "dev-empty")
	if err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected nil pending, got %+v", pending)
	}
}

func TestGetLaunchRequestPollDoesNotEnrich(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-manifest"})
	_, _ = svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-manifest", UserID: "user-manifest"})
	owner := Owner{UserID: "user-manifest"}
	inst, _ := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, _ := svc.CreateLaunchRequest(ctx, owner, CreateLaunchRequestInput{
		InstanceID: inst.ID, DeviceID: "dev-manifest",
	})

	got, err := svc.GetLaunchRequest(ctx, owner, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.LaunchStatusQueued {
		t.Fatalf("expected queued without side effects, got status=%q error=%v", got.Status, got.ErrorCode)
	}
}

func TestLicensedLaunchSkipsMojangRefreshAfterDispatch(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	stub := &stubMojangLauncher{
		linked: true,
		session: &mojang.SessionView{
			Username:    "Notch",
			UUID:        "uuid",
			AccessToken: "token",
		},
	}
	svc.SetMojang(stub)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-mojang-skip"})
	_, _ = svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-mojang-skip", UserID: "user-mojang-skip"})
	owner := Owner{UserID: "user-mojang-skip"}
	inst, _ := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, err := svc.CreateLaunchRequest(ctx, owner, CreateLaunchRequestInput{
		InstanceID:       inst.ID,
		DeviceID:         "dev-mojang-skip",
		UseMojangAccount: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.FetchPendingLaunch(ctx, "dev-mojang-skip"); err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if stub.sessionCalls != 1 {
		t.Fatalf("expected one session refresh on dispatch, got %d", stub.sessionCalls)
	}

	updated, err := svc.UpdateLaunchRequest(ctx, "dev-mojang-skip", created.ID, UpdateLaunchRequestInput{
		Status: models.LaunchStatusRunning,
		PID:    intPtr(4242),
	})
	if err != nil {
		t.Fatalf("update running: %v", err)
	}
	if updated.Status != models.LaunchStatusRunning {
		t.Fatalf("status: %+v", updated)
	}
	if stub.sessionCalls != 1 {
		t.Fatalf("running poll must not refresh mojang session, calls=%d", stub.sessionCalls)
	}

	got, err := svc.GetLaunchRequest(ctx, owner, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.LaunchStatusRunning {
		t.Fatalf("expected running, got %+v", got)
	}
	if stub.sessionCalls != 1 {
		t.Fatalf("get running must not refresh mojang session, calls=%d", stub.sessionCalls)
	}
}

func TestLicensedLaunchMojangRevokedMarksFailed(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	svc.SetMojang(&stubMojangLauncher{
		linked:     true,
		sessionErr: fmt.Errorf("%w: invalid_grant", mojang.ErrSessionRevoked),
	})
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-mojang-revoked"})
	_, _ = svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-mojang-revoked", UserID: "user-mojang-revoked"})
	owner := Owner{UserID: "user-mojang-revoked"}
	inst, _ := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, _ := svc.CreateLaunchRequest(ctx, owner, CreateLaunchRequestInput{
		InstanceID:       inst.ID,
		DeviceID:         "dev-mojang-revoked",
		UseMojangAccount: true,
	})

	pending, err := svc.FetchPendingLaunch(ctx, "dev-mojang-revoked")
	if err != nil {
		t.Fatalf("fetch pending: %v", err)
	}
	if pending != nil {
		t.Fatalf("expected nil pending after revoked session, got %+v", pending)
	}

	got, err := svc.GetLaunchRequest(ctx, owner, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.LaunchStatusFailed || got.ErrorCode == nil || *got.ErrorCode != "MOJANG_SESSION" {
		t.Fatalf("expected MOJANG_SESSION failure, got status=%q error=%v", got.Status, got.ErrorCode)
	}
}

func TestLicensedLaunchMojangUnavailableDoesNotFail(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	svc.SetMojang(&stubMojangLauncher{
		linked:     true,
		sessionErr: fmt.Errorf("%w: timeout", mojang.ErrSessionUnavailable),
	})
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-mojang-up"})
	_, _ = svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-mojang-up", UserID: "user-mojang-up"})
	owner := Owner{UserID: "user-mojang-up"}
	inst, _ := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Survival", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	created, _ := svc.CreateLaunchRequest(ctx, owner, CreateLaunchRequestInput{
		InstanceID:       inst.ID,
		DeviceID:         "dev-mojang-up",
		UseMojangAccount: true,
	})

	_, err := svc.FetchPendingLaunch(ctx, "dev-mojang-up")
	if !errors.Is(err, ErrMojangUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}

	got, err := svc.GetLaunchRequest(ctx, owner, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != models.LaunchStatusQueued {
		t.Fatalf("transient mojang failure must revert to queued, got status=%q", got.Status)
	}
}

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

func TestCreateLaunchRequestWithJoinServer(t *testing.T) {
	svc, db, _ := newLauncherService(t)
	ctx := context.Background()

	_, _ = svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-join"})
	_, _ = svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-join", UserID: "user-join"})
	owner := Owner{UserID: "user-join"}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name: "Client", MCVersion: "1.21", Loader: models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("instance: %v", err)
	}

	created, err := svc.CreateLaunchRequest(ctx, owner, CreateLaunchRequestInput{
		InstanceID:        inst.ID,
		DeviceID:          "dev-join",
		JoinServerAddress: "play.example.com",
		JoinServerPort:    25565,
	})
	if err != nil {
		t.Fatalf("create launch: %v", err)
	}
	if created.JoinServerAddress == nil || *created.JoinServerAddress != "play.example.com" {
		t.Fatalf("join address: %+v", created.JoinServerAddress)
	}
	if created.JoinServerPort == nil || *created.JoinServerPort != 25565 {
		t.Fatalf("join port: %+v", created.JoinServerPort)
	}

	var stored models.LaunchRequest
	if err := db.Where("id = ?", created.ID).First(&stored).Error; err != nil {
		t.Fatalf("load stored: %v", err)
	}
	if stored.JoinServerAddress == nil || *stored.JoinServerAddress != "play.example.com" {
		t.Fatalf("stored address: %+v", stored.JoinServerAddress)
	}
}

