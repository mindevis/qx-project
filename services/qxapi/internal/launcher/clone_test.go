package launcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestCloneDisplayName(t *testing.T) {
	if got := cloneDisplayName("Survival"); got != "Survival (copy)" {
		t.Fatalf("got %q", got)
	}
	if got := cloneDisplayName("  "); got != "Instance (copy)" {
		t.Fatalf("blank: %q", got)
	}
}

func TestCloneInstanceRequiresLinkedDevice(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}
	created, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:      "Survival",
		MCVersion: "1.21",
		Loader:    models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.CloneInstance(ctx, owner, created.Instance.ID)
	if !errors.Is(err, ErrDeviceNotLinked) {
		t.Fatalf("expected device not linked, got %v", err)
	}
	items, err := svc.ListInstances(ctx, owner)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected only source instance, got %d", len(items))
	}
}

func TestCloneInstanceCopiesMetadataAndFiles(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}
	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-clone"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-clone", UserID: owner.UserID}); err != nil {
		t.Fatalf("link: %v", err)
	}

	created, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Modded",
		MCVersion:     "1.20.1",
		Loader:        models.LoaderFabric,
		LoaderVersion: "0.16.14",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	src := created.Instance
	gsID := "gs-bound"
	src.ManagedByGameServerID = &gsID
	src.Mods = models.InstanceResourceList{
		{Source: "modrinth", ProjectID: "sodium", Filename: "sodium.jar", ResourceType: "mod"},
	}
	src.ExtraJVMArgs = models.StringList{"-XX:+UseG1GC"}
	if err := svc.db.WithContext(ctx).Save(src).Error; err != nil {
		t.Fatalf("save src: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			pending, err := svc.FetchPendingInstanceFile(ctx, "dev-clone")
			if err != nil {
				done <- err
				return
			}
			if pending != nil && pending.Operation == models.InstanceFileOpClone {
				if pending.Path != src.ID {
					done <- errors.New("clone path should be source instance id")
					return
				}
				_, err := svc.UpdateInstanceFileRequest(ctx, "dev-clone", pending.ID, UpdateInstanceFileRequestInput{
					Status:     models.InstanceFileStatusCompleted,
					ResultJSON: `{"copied":true}`,
				})
				done <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- errors.New("timed out waiting for clone file request")
	}()

	result, err := svc.CloneInstance(ctx, owner, src.ID)
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("file bridge: %v", err)
	}
	if result.Instance.ID == src.ID {
		t.Fatal("clone should get a new id")
	}
	if result.Instance.Name != "Modded (copy)" {
		t.Fatalf("name: %q", result.Instance.Name)
	}
	if result.Instance.ManagedByGameServerID != nil {
		t.Fatal("clone must not keep game server binding")
	}
	if result.Instance.MCVersion != "1.20.1" || result.Instance.Loader != models.LoaderFabric {
		t.Fatalf("version/loader: %+v", result.Instance)
	}
	if len(result.Instance.Mods) != 1 || result.Instance.Mods[0].Filename != "sodium.jar" {
		t.Fatalf("mods: %+v", result.Instance.Mods)
	}
	if len(result.Instance.ExtraJVMArgs) != 1 || result.Instance.ExtraJVMArgs[0] != "-XX:+UseG1GC" {
		t.Fatalf("jvm args: %+v", result.Instance.ExtraJVMArgs)
	}
	if result.PrepareRequestID != nil {
		t.Fatal("prepare should be skipped when files were copied")
	}
}

func TestCloneInstanceEnqueuesPrepareWhenFilesMissing(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}
	if _, err := svc.RegisterDevice(ctx, RegisterDeviceInput{DeviceID: "dev-clone-prep"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.LinkDevice(ctx, LinkDeviceInput{DeviceID: "dev-clone-prep", UserID: owner.UserID}); err != nil {
		t.Fatalf("link: %v", err)
	}
	created, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:      "Empty",
		MCVersion: "1.21",
		Loader:    models.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			pending, err := svc.FetchPendingInstanceFile(ctx, "dev-clone-prep")
			if err != nil {
				done <- err
				return
			}
			if pending != nil && pending.Operation == models.InstanceFileOpClone {
				_, err := svc.UpdateInstanceFileRequest(ctx, "dev-clone-prep", pending.ID, UpdateInstanceFileRequestInput{
					Status:     models.InstanceFileStatusCompleted,
					ResultJSON: `{"copied":false}`,
				})
				done <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		done <- errors.New("timed out waiting for clone file request")
	}()

	result, err := svc.CloneInstance(ctx, owner, created.Instance.ID)
	if err != nil {
		t.Fatalf("clone: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("file bridge: %v", err)
	}
	if result.PrepareRequestID == nil || *result.PrepareRequestID == "" {
		t.Fatal("expected prepare request when files were not copied")
	}
}
