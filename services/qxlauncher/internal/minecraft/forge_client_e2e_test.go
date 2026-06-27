package minecraft

import (
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

// Full Forge client launch for manual verification.
//
// Run:
//
//	make test-forge-client
//
// Or:
//
//	QX_FORGE_E2E=1 go test ./internal/minecraft -run TestIntegrationForgeClientLaunch -count=1 -timeout 30m -v
//
// Keep the game open until you close it:
//
//	QX_FORGE_E2E=1 QX_FORGE_E2E_KEEP=1 make test-forge-client
func TestIntegrationForgeClientLaunch(t *testing.T) {
	runClientLaunchE2E(t, clientLaunchE2EOptions{
		EnvFlag:         "QX_FORGE_E2E",
		MCVersion:       "1.20.1",
		LoaderVersion:   "47.4.20",
		Loader:          mcmanifest.LoaderForge,
		InstanceID:      "forge-e2e",
		DataDirEnv:      "QX_FORGE_E2E_DATA",
		DataDirDefault:  "qx-forge-e2e",
		UsernameEnv:     "QX_FORGE_USERNAME",
		UsernameDefault: "ForgeTest",
		AliveEnv:        "QX_FORGE_E2E_ALIVE",
		KeepEnv:         "QX_FORGE_E2E_KEEP",
		LogReady: func(path string) bool {
			return launchLogShowsModLoaderReady(path, "MinecraftForge v")
		},
		LoaderLabel: "Forge",
	})
}
