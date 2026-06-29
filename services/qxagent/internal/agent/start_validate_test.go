package agent

import (
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestSanitizeStartPayloadJarOnly(t *testing.T) {
	dir := t.TempDir()
	jar := filepath.Join(dir, "server.jar")
	payload := protocol.ServerStartPayload{WorkDir: dir, JarPath: jar}
	if err := sanitizeStartPayload(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.JarPath != jar {
		t.Fatalf("jar: %q", payload.JarPath)
	}
}

func TestSanitizeStartPayloadRejectsShellMetacharCommand(t *testing.T) {
	dir := t.TempDir()
	payload := protocol.ServerStartPayload{
		WorkDir: dir,
		Command: "rm",
	}
	if err := sanitizeStartPayload(&payload); err == nil {
		t.Fatal("expected disallowed command")
	}
}

func TestSanitizeStartPayloadAllowsJava(t *testing.T) {
	dir := t.TempDir()
	payload := protocol.ServerStartPayload{
		WorkDir: dir,
		Command: "java",
		Args:    []string{"-version"},
	}
	if err := sanitizeStartPayload(&payload); err != nil {
		t.Fatal(err)
	}
}
