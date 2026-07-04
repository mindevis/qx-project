package minecraft

import (
	"context"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type fixedManifestBuilder struct {
	manifest *mcmanifest.InstanceLaunchManifest
}

func (f fixedManifestBuilder) BuildInstanceManifest(_ context.Context, instanceID, name, mcVersion, loader, loaderVersion, targetOS string) (*mcmanifest.InstanceLaunchManifest, error) {
	m := *f.manifest
	if m.InstanceID == "" {
		m.InstanceID = instanceID
	}
	return &m, nil
}

func TestBuildLaunchManifestUsesBuilder(t *testing.T) {
	want := &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		MCVersion:  "1.21",
		MainClass:  "Main",
	}
	dl := NewDownloader(t.TempDir())
	dl.ManifestBuilder = fixedManifestBuilder{manifest: want}

	got, err := dl.BuildLaunchManifest(context.Background(), LaunchInstance{
		ID:        "inst-1",
		MCVersion: "1.21",
		Loader:    mcmanifest.LoaderVanilla,
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if got.MainClass != want.MainClass || got.InstanceID != want.InstanceID {
		t.Fatalf("unexpected manifest: %+v", got)
	}
}

func TestApplyLaunchSettingsDefaultMemory(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		JVMArguments: []string{"-Xmx2G"},
	}
	applyLaunchSettings(manifest, LaunchInstance{})
	if manifest.JVMArguments[0] != "-Xms4G" || manifest.JVMArguments[1] != "-Xmx4G" {
		t.Fatalf("expected 4G defaults, got %v", manifest.JVMArguments)
	}
}
