package installer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		ServerType: "paper",
		WorkDir:    "/tmp",
		MCVersion:  "1.21",
		Port:       25565,
	})
	if !errors.Is(err, ErrUnsupportedServerType) {
		t.Fatalf("expected unsupported: %v", err)
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
