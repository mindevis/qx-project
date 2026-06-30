package agent

import (
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestValidateStartPayloadJarOnly(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "server.jar")
	payload := protocol.ServerStartPayload{WorkDir: dir, JarPath: jar}
	start, err := ValidateStartPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if start.JarPath != jar {
		t.Fatalf("jar: %q", start.JarPath)
	}
}

func TestValidateStartPayloadRejectsShellMetacharCommand(t *testing.T) {
	dir := t.TempDir()
	payload := protocol.ServerStartPayload{
		WorkDir: dir,
		Command: "rm",
	}
	if _, err := ValidateStartPayload(payload); err == nil {
		t.Fatal("expected disallowed command")
	}
}

func TestValidateStartPayloadAllowsJava(t *testing.T) {
	dir := t.TempDir()
	payload := protocol.ServerStartPayload{
		WorkDir: dir,
		Command: "java",
		Args:    []string{"-version"},
	}
	if _, err := ValidateStartPayload(payload); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStartPayloadAllowsMojangJavaOutsideWorkDir(t *testing.T) {
	dir := t.TempDir()
	javaBin := filepath.Join(filepath.Dir(dir), "java", "bin", "java")
	payload := protocol.ServerStartPayload{
		WorkDir: dir,
		Command: javaBin,
		JavaBin: javaBin,
		Args:    []string{"@user_jvm_args.txt", "@libraries/net/minecraftforge/forge/1.20.1-47.4.20/unix_args.txt", "nogui"},
	}
	start, err := ValidateStartPayload(payload)
	if err != nil {
		t.Fatalf("forge-style start with external java: %v", err)
	}
	if start.Command != javaBin || start.JavaBin != javaBin {
		t.Fatalf("java paths: command=%q java_bin=%q", start.Command, start.JavaBin)
	}
}
