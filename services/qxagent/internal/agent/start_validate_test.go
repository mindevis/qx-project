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
