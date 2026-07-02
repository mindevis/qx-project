package minecraft

import (
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestBuildLaunchPlanForgeModulePath(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		Loader:    mcmanifest.LoaderForge,
		MCVersion: "1.20.1",
		VersionID: "1.20.1-forge-47.4.20",
		MainClass: "cpw.mods.bootstraplauncher.BootstrapLauncher",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.20.1"},
		JVMArguments: []string{
			"-DlibraryDirectory=${library_directory}",
			"-p",
			"${library_directory}/cpw/mods/bootstraplauncher/1.1.2/bootstraplauncher-1.1.2.jar",
		},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--launchTarget", "forgeclient",
		},
	}
	plan := BuildLaunchPlan(
		manifest,
		`C:\game\versions\1.20.1-forge-47.4.20\1.20.1-forge-47.4.20.jar`,
		nil,
		`C:\game\natives`,
		`C:\game\assets`,
		`C:\game`,
		`C:\libs`,
		"Devis",
		"uuid-1",
		"",
		nil,
		"",
	)
	joined := strings.Join(plan.Args, " ")
	if strings.Contains(joined, "-cp") {
		t.Fatalf("forge must not use -cp: %s", joined)
	}
	if !strings.Contains(joined, `C:/libs`) || !strings.Contains(joined, "Devis") {
		t.Fatalf("substitutions missing: %s", joined)
	}
	if !strings.Contains(joined, "forgeclient") {
		t.Fatalf("game args missing: %s", joined)
	}
}

func TestBuildLaunchPlanForgeDirectConnect(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		Loader:    mcmanifest.LoaderForge,
		MCVersion: "1.20.1",
		VersionID: "1.20.1-forge-47.4.20",
		MainClass: "cpw.mods.bootstraplauncher.BootstrapLauncher",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.20.1"},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--launchTarget", "forgeclient",
		},
	}
	plan := BuildLaunchPlan(
		manifest,
		`C:\game\versions\1.20.1-forge-47.4.20\1.20.1-forge-47.4.20.jar`,
		nil,
		`C:\game\natives`,
		`C:\game\assets`,
		`C:\game`,
		`C:\libs`,
		"Devis",
		"uuid-1",
		"",
		nil,
		"mc.example.com:25565",
	)
	joined := strings.Join(plan.Args, " ")
	if !strings.Contains(joined, "--quickPlayMultiplayer mc.example.com:25565") {
		t.Fatalf("forge quick play connect missing: %s", joined)
	}
	idxQuick := strings.Index(joined, "--quickPlayMultiplayer")
	idxLaunch := strings.Index(joined, "--launchTarget")
	if idxQuick < 0 || idxLaunch < 0 || idxQuick > idxLaunch {
		t.Fatalf("join args must precede --launchTarget: %s", joined)
	}
}

func TestSubstituteLaunchArgEmbedded(t *testing.T) {
	got := substituteLaunchArg("-DignoreList=${version_name}.jar", map[string]string{
		"${version_name}": "1.20.1-forge-47.4.20",
	})
	if got != "-DignoreList=1.20.1-forge-47.4.20.jar" {
		t.Fatalf("got %q", got)
	}
}
