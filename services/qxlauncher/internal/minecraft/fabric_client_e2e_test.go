package minecraft

import (
	"context"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

// Full Fabric client launch for manual verification.
//
// Run:
//
//	make test-fabric-client
func TestIntegrationFabricClientLaunch(t *testing.T) {
	runClientLaunchE2E(t, clientLaunchE2EOptions{
		EnvFlag:               "QX_FABRIC_E2E",
		MCVersion:             "1.21.1",
		LoaderVersion:         "",
		LoaderVersionFallback: "0.19.3",
		Loader:                mcmanifest.LoaderFabric,
		InstanceID:            "fabric-e2e",
		DataDirEnv:            "QX_FABRIC_E2E_DATA",
		DataDirDefault:        "qx-fabric-e2e",
		UsernameEnv:           "QX_FABRIC_USERNAME",
		UsernameDefault:       "FabricTest",
		AliveEnv:              "QX_FABRIC_E2E_ALIVE",
		KeepEnv:               "QX_FABRIC_E2E_KEEP",
		ResolveLatest: func(ctx context.Context, mcVersion string) (string, error) {
			return mcmanifest.NewClient().ResolveLatestFabricLoaderVersion(ctx, mcVersion)
		},
		LogReady: func(path string) bool {
			return launchLogContains(path, "Fabric Loader", "Loading for game Minecraft")
		},
		LoaderLabel: "Fabric",
	})
}
