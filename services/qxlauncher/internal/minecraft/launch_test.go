package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestEnsureClientJar(t *testing.T) {
	dir := t.TempDir()
	body := []byte("fake-jar")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	manifest := &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		MCVersion:  "1.21",
		MainClass:  "net.minecraft.client.main.Main",
		ClientJar:  mcmanifest.DownloadFile{URL: srv.URL, Sha1: ""},
	}
	dl := NewDownloader(dir)
	path, err := dl.EnsureClientJar(context.Background(), manifest)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing jar: %v", err)
	}
}

func TestSanitizeLaunchArgs(t *testing.T) {
	args := sanitizeLaunchArgs([]string{
		"--username", "Steve",
		"--clientId", "",
		"--xuid", "",
		"--quickPlayPath", "",
		"${unresolved}",
	})
	if len(args) != 2 || args[0] != "--username" || args[1] != "Steve" {
		t.Fatalf("sanitized: %v", args)
	}
}

func TestExcludedFromLegacyClasspath(t *testing.T) {
	cases := map[string]bool{
		`C:\libs\net\neoforged\neoforge\21.1.234\neoforge-21.1.234-universal.jar`: true,
		`C:\libs\net\neoforged\neoforge\21.1.234\neoforge-21.1.234-client.jar`:    true,
		`C:\libs\cpw\mods\modlauncher\11.0.5\modlauncher-11.0.5.jar`:              false,
	}
	for path, want := range cases {
		if got := excludedFromLegacyClasspath(path); got != want {
			t.Fatalf("%q: got %v want %v", path, got, want)
		}
	}
}

func TestBuildLaunchPlanForgeLegacyClasspath(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion:  "1.20.1",
		Loader:     mcmanifest.LoaderForge,
		MainClass:  "cpw.mods.bootstraplauncher.BootstrapLauncher",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.20.1"},
		JVMArguments: []string{
			"-DlibraryDirectory=/libs",
			"-p", "bootstrap.jar",
		},
		GameArguments: []string{"--launchTarget", "forgeclient"},
	}
	libs := []string{`C:\libs\cpw\mods\modlauncher\10.0.9\modlauncher-10.0.9.jar`}
	plan := BuildLaunchPlan(manifest, "", libs, "", "/assets", "/game", "/libs", "Steve", "uuid-1", "", nil, "")
	joined := strings.Join(plan.Args, " ")
	if !strings.Contains(joined, "-DlegacyClassPath=") {
		t.Fatalf("missing legacy classpath: %s", joined)
	}
	if !strings.Contains(joined, "modlauncher-10.0.9.jar") {
		t.Fatalf("legacy classpath missing modlauncher: %s", joined)
	}
}

func TestBuildLaunchPlanModded(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion:  "1.20.1",
		MainClass:  "net.fabricmc.loader.impl.launch.knot.KnotClient",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.20.1"},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--gameDir", "${game_directory}",
		},
		JVMArguments: []string{"-Xmx4G"},
	}
	jar := filepath.Join(t.TempDir(), "1.20.1.jar")
	plan := BuildLaunchPlan(manifest, jar, []string{"/lib/a.jar"}, "/natives", "/assets", "/game", "/libs", "Steve", "uuid-1", "", nil, "")
	if plan.Args[0] != "-Xmx4G" {
		t.Fatalf("jvm args: %+v", plan.Args)
	}
	joined := strings.Join(plan.Args, " ")
	if !strings.Contains(joined, "Steve") || !strings.Contains(joined, "/game") {
		t.Fatalf("substituted args: %s", joined)
	}
}

func TestBuildLaunchPlanQuickPlayMultiplayer(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion: "1.21",
		MainClass: "net.minecraft.client.main.Main",
		GameArguments: []string{
			"--quickPlayMultiplayer", "${quickPlayMultiplayer}",
		},
	}
	jar := "/game/1.21.jar"
	plan := BuildLaunchPlan(manifest, jar, nil, "", "/assets", "/game", "/libs", "Steve", "uuid-1", "", nil, "play.example.com:25565")
	joined := strings.Join(plan.Args, " ")
	if !strings.Contains(joined, "play.example.com:25565") {
		t.Fatalf("quick play missing: %s", joined)
	}
}

func TestBuildLaunchPlanMinecraft26NativesPaths(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion:  "26.2",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "26.2"},
		JVMArguments: []string{
			"-Djava.library.path=${natives_directory}/java",
			"-Djna.tmpdir=${natives_directory}/jna",
			"-Dorg.lwjgl.system.SharedLibraryExtractPath=${natives_directory}/lwjgl",
			"-Dio.netty.native.workdir=${natives_directory}/netty",
			"-Dminecraft.launcher.brand=${launcher_name}",
			"-Dminecraft.launcher.version=${launcher_version}",
			"-cp",
			"${classpath}",
		},
		GameArguments: []string{
			"--username", "${auth_player_name}",
			"--version", "${version_name}",
			"--gameDir", "${game_directory}",
		},
	}
	jar := filepath.Join(t.TempDir(), "26.2.jar")
	lib := filepath.Join(t.TempDir(), "lwjgl.jar")
	natives := filepath.Join(t.TempDir(), "natives")
	if err := os.MkdirAll(filepath.Join(natives, "java"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(natives, "java", "lwjgl.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := BuildLaunchPlan(manifest, jar, []string{lib}, natives, "/assets", "/game", "/libs", "Steve", "uuid-1", "", nil, "")
	joined := strings.Join(plan.Args, " ")
	if strings.Contains(joined, "${") {
		t.Fatalf("unresolved placeholders: %s", joined)
	}
	wantLibPath := "-Djava.library.path=" + strings.ReplaceAll(natives, "\\", "/") + "/java"
	if !strings.Contains(joined, wantLibPath) {
		t.Fatalf("library path: %s", joined)
	}
	if !strings.Contains(joined, "QXLauncher") {
		t.Fatalf("launcher name: %s", joined)
	}
	if !strings.Contains(joined, "-Dorg.lwjgl.librarypath=") {
		t.Fatalf("missing lwjgl librarypath: %s", joined)
	}
	if !strings.Contains(joined, lib) || !strings.Contains(joined, jar) {
		t.Fatalf("classpath missing jars: %s", joined)
	}
	cpCount := 0
	for _, arg := range plan.Args {
		if arg == "-cp" || arg == "-classpath" {
			cpCount++
		}
	}
	if cpCount != 1 {
		t.Fatalf("expected one -cp, got %d: %s", cpCount, joined)
	}
}

func TestBuildLaunchPlanFallsBackWhenJavaNativesDirMissing(t *testing.T) {
	natives := t.TempDir()
	if err := os.WriteFile(filepath.Join(natives, "lwjgl.dll"), []byte("dll"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion:  "26.2",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "26.2"},
		JVMArguments: []string{
			"-Djava.library.path=${natives_directory}/java",
			"-Dminecraft.launcher.brand=${launcher_name}",
			"-cp",
			"${classpath}",
		},
	}
	jar := filepath.Join(t.TempDir(), "26.2.jar")
	plan := BuildLaunchPlan(manifest, jar, []string{"lib.jar"}, natives, "/assets", "/game", "/libs", "Steve", "uuid-1", "", nil, "")
	joined := strings.Join(plan.Args, " ")
	if strings.Contains(joined, "${") {
		t.Fatalf("unresolved placeholders: %s", joined)
	}
	wantLibPath := "-Djava.library.path=" + strings.ReplaceAll(natives, "\\", "/")
	if !strings.Contains(joined, wantLibPath) {
		t.Fatalf("expected natives root library path, got: %s", joined)
	}
	if strings.Contains(joined, wantLibPath+"/java") {
		t.Fatalf("should not keep missing java natives dir: %s", joined)
	}
}

func TestBuildLaunchPlan(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion:  "1.21",
		MainClass:  "net.minecraft.client.main.Main",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.21"},
	}
	jar := filepath.Join(t.TempDir(), "1.21.jar")
	plan := BuildLaunchPlan(manifest, jar, []string{"/lib/a.jar"}, "/natives", "/assets", "/game", "/libs", "Steve", "uuid-1", "", nil, "")
	if plan.MainClass == "" || len(plan.Args) == 0 {
		t.Fatalf("plan: %+v", plan)
	}
	if plan.WorkingDir != "/game" {
		t.Fatalf("game dir: %s", plan.WorkingDir)
	}
}

func TestBuildClasspath(t *testing.T) {
	cp := BuildClasspath([]string{"a.jar", "b.jar"})
	if cp == "" || !strings.Contains(cp, "a.jar") {
		t.Fatalf("classpath: %q", cp)
	}
}

func TestMavenRelPath(t *testing.T) {
	p := MavenRelPath("com.mojang:patchy:1.3.9")
	if !strings.Contains(p, "patchy-1.3.9.jar") {
		t.Fatalf("path: %s", p)
	}
}
