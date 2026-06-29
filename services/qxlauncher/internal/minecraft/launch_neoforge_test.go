package minecraft

import (
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestBuildLaunchPlanNeoForgeModulePath(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		Loader:    mcmanifest.LoaderNeoForge,
		MCVersion: "1.21.1",
		VersionID: "1.21.1-neoforge-21.1.234",
		MainClass: "cpw.mods.bootstraplauncher.BootstrapLauncher",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.21.1"},
		JVMArguments: []string{
			"-DlibraryDirectory=${library_directory}",
			"-p",
			"${library_directory}/cpw/mods/bootstraplauncher/2.0.2/bootstraplauncher-2.0.2.jar",
		},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--launchTarget", "forgeclient",
		},
	}
	libs := []string{
		`C:\libs\net\neoforged\neoforge\21.1.234\neoforge-21.1.234-universal.jar`,
		`C:\libs\net\neoforged\neoforge\21.1.234\neoforge-21.1.234-client.jar`,
		`C:\libs\cpw\mods\modlauncher\11.0.5\modlauncher-11.0.5.jar`,
	}
	plan := BuildLaunchPlan(
		manifest,
		`C:\game\versions\1.21.1-neoforge-21.1.234\1.21.1-neoforge-21.1.234.jar`,
		libs,
		`C:\game\natives`,
		`C:\game\assets`,
		`C:\game`,
		`C:\libs`,
		"NeoForgeTest",
		"uuid-1",
		"",
		nil,
	)
	joined := strings.Join(plan.Args, " ")
	if strings.Contains(joined, "-cp") {
		t.Fatalf("neoforge must not use -cp: %s", joined)
	}
	if !strings.Contains(joined, "-DlegacyClassPath=") {
		t.Fatalf("missing legacy classpath: %s", joined)
	}
	if strings.Contains(joined, "neoforge-21.1.234-universal.jar") || strings.Contains(joined, "neoforge-21.1.234-client.jar") {
		t.Fatalf("duplicate neoforge jars must be excluded from legacy classpath: %s", joined)
	}
	if !strings.Contains(joined, "modlauncher-11.0.5.jar") {
		t.Fatalf("legacy classpath missing modlauncher: %s", joined)
	}
	if !strings.Contains(joined, "forgeclient") {
		t.Fatalf("game args missing: %s", joined)
	}
}

func TestLoaderDisplayLabel(t *testing.T) {
	cases := map[string]string{
		mcmanifest.LoaderNeoForge: "NeoForge",
		mcmanifest.LoaderForge:    "Forge",
		mcmanifest.LoaderFabric:   "Fabric",
		mcmanifest.LoaderQuilt:    "Quilt",
		mcmanifest.LoaderVanilla:  "Vanilla",
		"":                        "Vanilla",
		"unknown":                 "unknown",
	}
	for loader, want := range cases {
		if got := LoaderDisplayLabel(loader); got != want {
			t.Fatalf("%q: got %q want %q", loader, got, want)
		}
	}
}
