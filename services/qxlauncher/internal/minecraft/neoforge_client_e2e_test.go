package minecraft

import (
	"context"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

// Full NeoForge client launch for manual verification.
//
// Run:
//
//	make test-neoforge-client
//
// Or:
//
//	QX_NEOFORGE_E2E=1 go test ./internal/minecraft -run TestIntegrationNeoForgeClientLaunch -count=1 -timeout 30m -v
//
// Keep the game open until you close it:
//
//	QX_NEOFORGE_E2E=1 QX_NEOFORGE_E2E_KEEP=1 make test-neoforge-client
func TestIntegrationNeoForgeClientLaunch(t *testing.T) {
	runClientLaunchE2E(t, clientLaunchE2EOptions{
		EnvFlag:               "QX_NEOFORGE_E2E",
		MCVersion:             "1.21.1",
		LoaderVersion:         "",
		LoaderVersionFallback: "21.1.234",
		Loader:                mcmanifest.LoaderNeoForge,
		InstanceID:      "neoforge-e2e",
		DataDirEnv:      "QX_NEOFORGE_E2E_DATA",
		DataDirDefault:  "qx-neoforge-e2e",
		UsernameEnv:     "QX_NEOFORGE_USERNAME",
		UsernameDefault: "NeoForgeTest",
		AliveEnv:        "QX_NEOFORGE_E2E_ALIVE",
		KeepEnv:         "QX_NEOFORGE_E2E_KEEP",
		ResolveLatest: func(ctx context.Context, mcVersion string) (string, error) {
			return mcmanifest.NewClient().ResolveLatestNeoForgeVersion(ctx, mcVersion)
		},
		LogReady: func(path string) bool {
			return launchLogShowsModLoaderReady(path, "NeoForge", "neoforged", "MinecraftForge")
		},
		LoaderLabel: "NeoForge",
	})
}
