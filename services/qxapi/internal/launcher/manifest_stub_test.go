package launcher

import (
	"context"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type stubManifestProvider struct {
	manifest *mcmanifest.InstanceLaunchManifest
	err      error
}

func (s stubManifestProvider) BuildInstanceManifest(_ context.Context, instanceID, name, mcVersion, loader, loaderVersion string) (*mcmanifest.InstanceLaunchManifest, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.manifest != nil {
		return s.manifest, nil
	}
	return &mcmanifest.InstanceLaunchManifest{
		InstanceID: instanceID,
		Name:       name,
		MCVersion:  mcVersion,
		Loader:     loader,
		MainClass:  "net.minecraft.client.main.Main",
		ClientJar:  mcmanifest.DownloadFile{URL: "https://example/client.jar", Sha1: "abc", Size: 1},
	}, nil
}

func (s *Service) withStubManifest(t testing.TB, m *mcmanifest.InstanceLaunchManifest) {
	t.Helper()
	s.SetManifestProvider(stubManifestProvider{manifest: m})
}
