package agent

import (
	"os"
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

func TestValidateStartPayloadRejectsShellMetacharArgs(t *testing.T) {
	dir := t.TempDir()
	payload := protocol.ServerStartPayload{
		WorkDir:   dir,
		Command:   "java",
		ExtraArgs: []string{";rm -rf /"},
	}
	if _, err := ValidateStartPayload(payload); err == nil {
		t.Fatal("expected disallowed extra arg")
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

func TestMergeCommandArgsDropsUserJVMFileWhenInline(t *testing.T) {
	t.Parallel()
	got := mergeCommandArgs(ValidatedStart{
		JVMArgs:   []string{"-Xms2G", "-Xmx2G"},
		Args:      []string{"@user_jvm_args.txt", "@libraries/unix_args.txt", "nogui"},
		ExtraArgs: []string{"--forceUpgrade"},
	})
	want := []string{"-Xms2G", "-Xmx2G", "@libraries/unix_args.txt", "nogui", "--forceUpgrade"}
	if len(got) != len(want) {
		t.Fatalf("args: %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args: %+v", got)
		}
	}
}

func TestWriteUserJVMArgsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := writeUserJVMArgsFile(dir, []string{"-Xms1G", "-Xmx2G"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "user_jvm_args.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "-Xms1G\n-Xmx2G\n" {
		t.Fatalf("file: %q", got)
	}
}
