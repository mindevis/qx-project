package minecraft

import (
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestBuildLaunchPlanFabricClasspath(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		Loader:        mcmanifest.LoaderFabric,
		MCVersion:     "1.21.1",
		LoaderVersion: "0.19.3",
		VersionID:     "fabric-loader-0.19.3-1.21.1",
		MainClass:     "net.fabricmc.loader.impl.launch.knot.KnotClient",
		AssetIndex:    mcmanifest.AssetIndexRef{ID: "1.21.1"},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--gameDir", "${game_directory}",
		},
		JVMArguments: []string{"-Xmx2G"},
	}
	libs := []string{`C:\libs\net\fabricmc\fabric-loader\0.19.3\fabric-loader-0.19.3.jar`}
	plan := BuildLaunchPlan(
		manifest,
		`C:\game\versions\fabric-loader-0.19.3-1.21.1\fabric-loader-0.19.3-1.21.1.jar`,
		libs,
		`C:\game\natives`,
		`C:\game\assets`,
		`C:\game`,
		`C:\libs`,
		"FabricTest",
		"uuid-1",
		"",
	)
	joined := strings.Join(plan.Args, " ")
	if !strings.Contains(joined, "-cp") {
		t.Fatalf("fabric must use classpath launch: %s", joined)
	}
	if strings.Contains(joined, "-DlegacyClassPath=") {
		t.Fatalf("fabric must not use legacy module classpath: %s", joined)
	}
	if plan.MainClass != "net.fabricmc.loader.impl.launch.knot.KnotClient" {
		t.Fatalf("main class: %s", plan.MainClass)
	}
	if !strings.Contains(joined, "FabricTest") {
		t.Fatalf("substitutions missing: %s", joined)
	}
}

func TestBuildLaunchPlanQuiltClasspath(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		Loader:        mcmanifest.LoaderQuilt,
		MCVersion:     "1.21.1",
		LoaderVersion: "0.28.1",
		VersionID:     "quilt-loader-0.28.1-1.21.1",
		MainClass:     "org.quiltmc.loader.impl.launch.knot.KnotClient",
		AssetIndex:    mcmanifest.AssetIndexRef{ID: "1.21.1"},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--gameDir", "${game_directory}",
		},
		JVMArguments: []string{"-Xmx2G"},
	}
	plan := BuildLaunchPlan(
		manifest,
		`C:\game\versions\quilt-loader-0.28.1-1.21.1\quilt-loader-0.28.1-1.21.1.jar`,
		[]string{`C:\libs\org\quiltmc\quilt-loader\0.28.1\quilt-loader-0.28.1.jar`},
		`C:\game\natives`,
		`C:\game\assets`,
		`C:\game`,
		`C:\libs`,
		"QuiltTest",
		"uuid-1",
		"",
	)
	joined := strings.Join(plan.Args, " ")
	if !strings.Contains(joined, "-cp") {
		t.Fatalf("quilt must use classpath launch: %s", joined)
	}
	if plan.MainClass != "org.quiltmc.loader.impl.launch.knot.KnotClient" {
		t.Fatalf("main class: %s", plan.MainClass)
	}
}

func TestFormatLaunchLabelFabricQuilt(t *testing.T) {
	fabric := FormatLaunchLabel(&mcmanifest.InstanceLaunchManifest{
		MCVersion:     "1.21.1",
		Loader:        mcmanifest.LoaderFabric,
		LoaderVersion: "0.19.3",
	}, "FabricTest")
	if fabric != "FabricTest · 1.21.1 · Fabric · 0.19.3" {
		t.Fatalf("fabric label: %q", fabric)
	}
	quilt := FormatLaunchSummary(&mcmanifest.InstanceLaunchManifest{
		MCVersion:     "1.21.1",
		Loader:        mcmanifest.LoaderQuilt,
		LoaderVersion: "0.28.1",
	})
	if quilt != "Quilt 0.28.1 · MC 1.21.1" {
		t.Fatalf("quilt summary: %q", quilt)
	}
}
