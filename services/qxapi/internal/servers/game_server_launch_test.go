package servers

import (
	"reflect"
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestGameServerStartArgSetsPaperJar(t *testing.T) {
	t.Parallel()
	minRAM, maxRAM := 1024, 2048
	item := &models.GameServer{
		JarPath:       "/opt/qxsystem/server/instances/gs-1/server.jar",
		StartArgsJSON: `["nogui"]`,
		MinMemoryMB:   &minRAM,
		MaxMemoryMB:   &maxRAM,
		ExtraJVMArgs:  models.StringList{"-XX:+UseG1GC", "-Xmx8G"},
		ExtraArgs:     models.StringList{"--forceUpgrade"},
	}
	args, jvm, extra := gameServerStartArgSets(item)
	if args != nil {
		t.Fatalf("jar start should not use command args: %+v", args)
	}
	if !reflect.DeepEqual(jvm, []string{"-Xms1G", "-Xmx2G", "-XX:+UseG1GC"}) {
		t.Fatalf("jvm: %+v", jvm)
	}
	if !reflect.DeepEqual(extra, []string{"nogui", "--forceUpgrade"}) {
		t.Fatalf("extra: %+v", extra)
	}
}

func TestGameServerStartArgSetsForgeCommand(t *testing.T) {
	t.Parallel()
	item := &models.GameServer{
		StartCommand:  "/opt/qxsystem/java/bin/java",
		StartArgsJSON: `["@user_jvm_args.txt","@libraries/unix_args.txt","nogui"]`,
		ExtraArgs:     models.StringList{"--world", "test"},
	}
	args, jvm, extra := gameServerStartArgSets(item)
	if !reflect.DeepEqual(args, []string{"@user_jvm_args.txt", "@libraries/unix_args.txt", "nogui"}) {
		t.Fatalf("args: %+v", args)
	}
	if !reflect.DeepEqual(jvm, []string{"-Xms2G", "-Xmx2G"}) {
		t.Fatalf("default jvm: %+v", jvm)
	}
	if !reflect.DeepEqual(extra, []string{"--world", "test"}) {
		t.Fatalf("extra: %+v", extra)
	}
}

func TestSanitizeGameServerExecArgsRejectsShell(t *testing.T) {
	t.Parallel()
	if _, err := sanitizeGameServerExecArgs([]string{"-XX:+UseG1GC", ";rm -rf /"}); err == nil {
		t.Fatal("expected invalid exec arg")
	}
	clean, err := sanitizeGameServerExecArgs([]string{"  -XX:+UseG1GC  ", ""})
	if err != nil || len(clean) != 1 || clean[0] != "-XX:+UseG1GC" {
		t.Fatalf("clean: %+v %v", clean, err)
	}
}
