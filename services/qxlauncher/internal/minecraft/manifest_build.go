package minecraft

import (
	"context"
	"runtime"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

// ManifestBuilder builds a launch manifest for an instance. When nil on Downloader, the default Mojang/meta client is used.
type ManifestBuilder interface {
	BuildInstanceManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion, targetOS string) (*mcmanifest.InstanceLaunchManifest, error)
}

type LaunchInstance struct {
	ID            string
	Name          string
	MCVersion     string
	Loader        string
	LoaderVersion string
	MaxMemoryMB   int
}

func (d *Downloader) BuildLaunchManifest(ctx context.Context, inst LaunchInstance) (*mcmanifest.InstanceLaunchManifest, error) {
	targetOS := runtime.GOOS
	var manifest *mcmanifest.InstanceLaunchManifest
	var err error
	if d.ManifestBuilder != nil {
		manifest, err = d.ManifestBuilder.BuildInstanceManifest(ctx, inst.ID, inst.Name, inst.MCVersion, inst.Loader, inst.LoaderVersion, targetOS)
	} else {
		manifest, err = mcmanifest.NewClient().BuildInstanceManifest(ctx, inst.ID, inst.Name, inst.MCVersion, inst.Loader, inst.LoaderVersion, targetOS)
	}
	if err != nil {
		return nil, err
	}
	if inst.MaxMemoryMB > 0 {
		mcmanifest.ApplyMaxMemoryMB(manifest, inst.MaxMemoryMB)
	}
	return manifest, nil
}
