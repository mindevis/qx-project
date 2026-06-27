package minecraft

import (
	"context"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

// Full Quilt client launch for manual verification.
//
// Run:
//
//	make test-quilt-client
func TestIntegrationQuiltClientLaunch(t *testing.T) {
	runClientLaunchE2E(t, clientLaunchE2EOptions{
		EnvFlag:               "QX_QUILT_E2E",
		MCVersion:             "1.21.1",
		LoaderVersion:         "",
		LoaderVersionFallback: "0.28.1",
		Loader:                mcmanifest.LoaderQuilt,
		InstanceID:            "quilt-e2e",
		DataDirEnv:            "QX_QUILT_E2E_DATA",
		DataDirDefault:        "qx-quilt-e2e",
		UsernameEnv:           "QX_QUILT_USERNAME",
		UsernameDefault:       "QuiltTest",
		AliveEnv:              "QX_QUILT_E2E_ALIVE",
		KeepEnv:               "QX_QUILT_E2E_KEEP",
		ResolveLatest: func(ctx context.Context, mcVersion string) (string, error) {
			return mcmanifest.NewClient().ResolveLatestQuiltLoaderVersion(ctx, mcVersion)
		},
		LogReady: func(path string) bool {
			return launchLogContains(path, "Quilt Loader", "Loading for game Minecraft")
		},
		LoaderLabel: "Quilt",
	})
}
