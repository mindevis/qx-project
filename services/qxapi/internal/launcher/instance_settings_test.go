package launcher

import (
	"context"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type fixedManifestProvider struct{}

func (fixedManifestProvider) BuildInstanceManifest(_ context.Context, instanceID, name, mcVersion, loader, loaderVersion, _ string) (*mcmanifest.InstanceLaunchManifest, error) {
	return &mcmanifest.InstanceLaunchManifest{
		InstanceID:     instanceID,
		Name:           name,
		MCVersion:      mcVersion,
		Loader:         loader,
		GameArguments:  []string{"--version", mcVersion},
		JVMArguments:   []string{},
		MainClass:      "net.minecraft.client.main.Main",
	}, nil
}

func TestUpdateInstanceMaxMemory(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Test",
		MCVersion:     "1.21.1",
		Loader:        "fabric",
		LoaderVersion: "0.16.9",
	})
	if err != nil {
		t.Fatal(err)
	}

	ram := 4096
	updated, err := svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MaxMemoryMB: &ram,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.MaxMemoryMB == nil || *updated.MaxMemoryMB != 4096 {
		t.Fatalf("expected 4096 MB, got %+v", updated.MaxMemoryMB)
	}

	bad := 128
	_, err = svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MaxMemoryMB: &bad,
	})
	if err != ErrValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateInstanceLaunchSettings(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Test",
		MCVersion:     "1.21.1",
		Loader:        "fabric",
		LoaderVersion: "0.16.9",
	})
	if err != nil {
		t.Fatal(err)
	}

	minRAM := 1024
	maxRAM := 4096
	width := 1280
	height := 720
	extra := []string{"-XX:+UseG1GC", "  ", "-Dfoo=bar"}
	updated, err := svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MinMemoryMB:  &minRAM,
		MaxMemoryMB:  &maxRAM,
		ExtraJVMArgs: &extra,
		WindowWidth:  &width,
		WindowHeight: &height,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.MinMemoryMB == nil || *updated.MinMemoryMB != 1024 {
		t.Fatalf("min memory: %+v", updated.MinMemoryMB)
	}
	if len(updated.ExtraJVMArgs) != 2 || updated.ExtraJVMArgs[0] != "-XX:+UseG1GC" {
		t.Fatalf("extra jvm args: %+v", updated.ExtraJVMArgs)
	}
	if updated.WindowWidth == nil || *updated.WindowWidth != 1280 {
		t.Fatalf("window width: %+v", updated.WindowWidth)
	}

	minHigh := 8192
	_, err = svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MinMemoryMB: &minHigh,
		MaxMemoryMB: &maxRAM,
	})
	if err != ErrValidation {
		t.Fatalf("expected min>max validation error, got %v", err)
	}

	_, err = svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{})
	if err != ErrValidation {
		t.Fatalf("expected empty update validation error, got %v", err)
	}
}

func TestInstanceManifestAppliesLaunchSettings(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	svc.SetManifestProvider(fixedManifestProvider{})
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Test",
		MCVersion:     "1.21.1",
		Loader:        models.LoaderVanilla,
	})
	if err != nil {
		t.Fatal(err)
	}

	minRAM := 1024
	maxRAM := 2048
	width := 800
	height := 600
	extra := []string{"-XX:+UseG1GC"}
	_, err = svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MinMemoryMB:  &minRAM,
		MaxMemoryMB:  &maxRAM,
		ExtraJVMArgs: &extra,
		WindowWidth:  &width,
		WindowHeight: &height,
	})
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := svc.InstanceManifest(ctx, owner, inst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.JVMArguments) < 3 {
		t.Fatalf("expected jvm args, got %v", manifest.JVMArguments)
	}
	foundXms, foundXmx, foundExtra := false, false, false
	for _, arg := range manifest.JVMArguments {
		switch arg {
		case "-Xms1G":
			foundXms = true
		case "-Xmx2G":
			foundXmx = true
		case "-XX:+UseG1GC":
			foundExtra = true
		}
	}
	if !foundXms || !foundXmx || !foundExtra {
		t.Fatalf("jvm args: %v", manifest.JVMArguments)
	}
	foundWidth, foundHeight := false, false
	for i := 0; i < len(manifest.GameArguments); i++ {
		if manifest.GameArguments[i] == "--width" && i+1 < len(manifest.GameArguments) && manifest.GameArguments[i+1] == "800" {
			foundWidth = true
		}
		if manifest.GameArguments[i] == "--height" && i+1 < len(manifest.GameArguments) && manifest.GameArguments[i+1] == "600" {
			foundHeight = true
		}
	}
	if !foundWidth || !foundHeight {
		t.Fatalf("game args: %v", manifest.GameArguments)
	}
}
