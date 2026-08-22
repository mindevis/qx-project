package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallForgeDryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "forge",
		WorkDir:       workDir,
		MCVersion:     "1.20.1",
		LoaderVersion: "47.4.20",
		Port:          25565,
		RconPassword:  "test-rcon",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if spec.WorkDir != workDir {
		t.Fatalf("work dir: %q", spec.WorkDir)
	}
	if spec.JarPath == "" {
		t.Fatal("expected jar path")
	}
}

func TestJavaRootFromServerRoot(t *testing.T) {
	if got := JavaRootFromServerRoot("/opt/qxsystem/server"); got != filepath.Join("/opt/qxsystem", "java") {
		t.Fatalf("got %q", got)
	}
	if got := JavaRootFromServerRoot(""); got != "/opt/qxsystem/java" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestInstallUnsupportedType(t *testing.T) {
	_, err := Install(context.Background(), Options{}, InstallConfig{
		ServerType: "magma",
		WorkDir:    "/tmp",
		MCVersion:  "1.21",
		Port:       25565,
	})
	if !errors.Is(err, ErrUnsupportedServerType) {
		t.Fatalf("expected unsupported: %v", err)
	}
}

func TestInstallPaperDryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "paper",
		WorkDir:       workDir,
		MCVersion:     "26.2",
		LoaderVersion: "112",
		Port:          25566,
		RconPassword:  "test-rcon",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if spec.WorkDir != workDir {
		t.Fatalf("work dir: %q", spec.WorkDir)
	}
	if !strings.HasSuffix(spec.JarPath, "server.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}
}

func TestVelocityJavaMajor(t *testing.T) {
	cases := map[string]int{
		"4.1.0-SNAPSHOT": 25,
		"4.0.0":          25,
		"3.4.0-SNAPSHOT": 21,
		"3.3.0-SNAPSHOT": 21,
		"":               21,
		"velocity":       21,
	}
	for version, want := range cases {
		if got := VelocityJavaMajor(version); got != want {
			t.Fatalf("%q: got %d want %d", version, got, want)
		}
	}
}

func TestInstallVelocityDryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "velocity",
		WorkDir:       workDir,
		MCVersion:     "3.4.0-SNAPSHOT",
		LoaderVersion: "550",
		Port:          25565,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if spec.WorkDir != workDir {
		t.Fatalf("work dir: %q", spec.WorkDir)
	}
	if !strings.HasSuffix(spec.JarPath, "server.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}
}

func TestInstallVelocity4DryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "velocity",
		WorkDir:       workDir,
		MCVersion:     "4.1.0-SNAPSHOT",
		LoaderVersion: "21",
		Port:          25565,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if spec.WorkDir != workDir {
		t.Fatalf("work dir: %q", spec.WorkDir)
	}
	if !strings.HasSuffix(spec.JarPath, "server.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}
}

func TestInstallWaterfallDryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "waterfall",
		WorkDir:       workDir,
		MCVersion:     "1.21",
		LoaderVersion: "589",
		Port:          25565,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !strings.HasSuffix(spec.JarPath, "server.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}
}

func TestInstallBungeeCordDryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "bungeecord",
		WorkDir:       workDir,
		MCVersion:     "latest",
		LoaderVersion: "2248",
		Port:          25565,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !strings.HasSuffix(spec.JarPath, "server.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}
}

func TestInstallVelocityValidation(t *testing.T) {
	_, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType: "velocity",
		WorkDir:    t.TempDir(),
		MCVersion:  "3.4.0-SNAPSHOT",
		Port:       25565,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestInstallPaperValidation(t *testing.T) {
	_, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType: "paper",
		WorkDir:    t.TempDir(),
		MCVersion:  "26.2",
		Port:       25565,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPaperJarURLFromBuilds(t *testing.T) {
	body := []byte(`[
		{"id":111,"downloads":{"server:default":{"url":"https://example.test/111.jar"}}},
		{"id":112,"downloads":{"server:default":{"url":"https://example.test/112.jar"}}}
	]`)
	got, err := paperJarURLFromBuilds(body, "112")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if got != "https://example.test/112.jar" {
		t.Fatalf("url: %q", got)
	}
	if _, err := paperJarURLFromBuilds(body, "999"); err == nil {
		t.Fatal("expected missing build")
	}
}

func TestForgeStartSpecPrefersJava(t *testing.T) {
	dir := t.TempDir()
	artifact := "1.20.1-47.4.20"
	unixDir := filepath.Join(dir, "libraries", "net", "minecraftforge", "forge", artifact)
	if err := os.MkdirAll(unixDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unixDir, "unix_args.txt"), []byte("-Dtest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user_jvm_args.txt"), []byte("-Xmx1G\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	spec, err := forgeStartSpec(dir, artifact, "/opt/qxsystem/java/bin/java")
	if err != nil {
		t.Fatalf("forgeStartSpec: %v", err)
	}
	if spec.Command != "/opt/qxsystem/java/bin/java" {
		t.Fatalf("command: %q", spec.Command)
	}
	if len(spec.Args) != 3 || spec.Args[0] != "@user_jvm_args.txt" {
		t.Fatalf("args: %v", spec.Args)
	}
}

func TestInstallForgeValidation(t *testing.T) {
	_, err := Install(context.Background(), Options{}, InstallConfig{
		ServerType: "forge",
		WorkDir:    "/tmp",
		Port:       25565,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestInstallNeoForgeDryRun(t *testing.T) {
	workDir := t.TempDir()
	spec, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "neoforge",
		WorkDir:       workDir,
		MCVersion:     "1.21.1",
		LoaderVersion: "21.1.234",
		Port:          25565,
		RconPassword:  "test-rcon",
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if spec.WorkDir != workDir {
		t.Fatalf("work dir: %q", spec.WorkDir)
	}
	if !strings.HasSuffix(spec.JarPath, "neoforge-21.1.234.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}
}

func TestInstallNeoForgeValidation(t *testing.T) {
	_, err := Install(context.Background(), Options{}, InstallConfig{
		ServerType: "neoforge",
		WorkDir:    "/tmp",
		Port:       25565,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestNeoForgeStartSpecPrefersJava(t *testing.T) {
	dir := t.TempDir()
	artifact := "21.1.234"
	unixDir := filepath.Join(dir, "libraries", "net", "neoforged", "neoforge", artifact)
	if err := os.MkdirAll(unixDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unixDir, "unix_args.txt"), []byte("-Dtest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user_jvm_args.txt"), []byte("-Xmx1G\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	unixRel := filepath.ToSlash(filepath.Join("libraries", "net", "neoforged", "neoforge", artifact, "unix_args.txt"))
	spec, err := loaderStartSpec(dir, unixRel, "neoforge-"+artifact+".jar", "/opt/qxsystem/java/bin/java")
	if err != nil {
		t.Fatalf("loaderStartSpec: %v", err)
	}
	if spec.Command != "/opt/qxsystem/java/bin/java" {
		t.Fatalf("command: %q", spec.Command)
	}
	if len(spec.Args) != 3 || spec.Args[1] != "@"+unixRel {
		t.Fatalf("args: %v", spec.Args)
	}
}

func TestNeoForgeInstallerURL(t *testing.T) {
	got := neoForgeInstallerURL("21.1.234")
	want := "https://maven.neoforged.net/releases/net/neoforged/neoforge/21.1.234/neoforge-21.1.234-installer.jar"
	if got != want {
		t.Fatalf("url: %q", got)
	}
}

func TestInstallNeoForgeRejectsUnsafeLoaderVersion(t *testing.T) {
	_, err := Install(context.Background(), Options{DryRun: true}, InstallConfig{
		ServerType:    "neoforge",
		WorkDir:       t.TempDir(),
		MCVersion:     "1.21.1",
		LoaderVersion: "../evil",
	})
	if err == nil {
		t.Fatal("expected invalid artifact")
	}
}

func TestLoaderStartSpecRunShAndJarFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := loaderStartSpec(dir, "missing/unix_args.txt", "server.jar", "/opt/qxsystem/java/bin/java")
	if err != nil {
		t.Fatalf("run.sh: %v", err)
	}
	if spec.Command != filepath.Join(dir, "run.sh") {
		t.Fatalf("run.sh command: %q", spec.Command)
	}

	empty := t.TempDir()
	if err := os.WriteFile(filepath.Join(empty, "server.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err = loaderStartSpec(empty, "missing/unix_args.txt", "server.jar", "")
	if err != nil {
		t.Fatalf("jar: %v", err)
	}
	if spec.JarPath != filepath.Join(empty, "server.jar") {
		t.Fatalf("jar path: %q", spec.JarPath)
	}

	missing := t.TempDir()
	if _, err := loaderStartSpec(missing, "missing/unix_args.txt", "server.jar", ""); err == nil {
		t.Fatal("expected missing jar")
	}
}
