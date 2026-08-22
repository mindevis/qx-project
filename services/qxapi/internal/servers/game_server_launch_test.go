package servers

import (
	"reflect"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestGameServerStartArgSetsPaperJar(t *testing.T) {
	t.Parallel()
	minRAM, maxRAM := 1024, 2048
	item := &models.GameServer{
		ServerType:    "paper",
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
	if !hasAikarDefaults(jvm, "-Xms1G", "-Xmx2G") {
		t.Fatalf("jvm: %+v", jvm)
	}
	if countJVMArgKey(jvm, "-XX:UseG1GC") != 1 {
		t.Fatalf("expected one UseG1GC: %+v", jvm)
	}
	if !reflect.DeepEqual(extra, []string{"nogui", "--forceUpgrade"}) {
		t.Fatalf("extra: %+v", extra)
	}
}

func TestGameServerStartArgSetsForgeCommand(t *testing.T) {
	t.Parallel()
	item := &models.GameServer{
		ServerType:    "forge",
		StartCommand:  "/opt/qxsystem/java/bin/java",
		StartArgsJSON: `["@user_jvm_args.txt","@libraries/unix_args.txt","nogui"]`,
		ExtraArgs:     models.StringList{"--world", "test"},
	}
	args, jvm, extra := gameServerStartArgSets(item)
	if !reflect.DeepEqual(args, []string{"@user_jvm_args.txt", "@libraries/unix_args.txt", "nogui"}) {
		t.Fatalf("args: %+v", args)
	}
	if !hasAikarDefaults(jvm, "-Xms2G", "-Xmx2G") {
		t.Fatalf("default jvm: %+v", jvm)
	}
	if !reflect.DeepEqual(extra, []string{"--world", "test"}) {
		t.Fatalf("extra: %+v", extra)
	}
}

func TestGameServerJVMArgsApplyVelocityDefaults(t *testing.T) {
	t.Parallel()
	item := &models.GameServer{ServerType: "velocity"}
	jvm := gameServerJVMArgs(item)
	if len(jvm) < 2 || jvm[0] != "-Xms2G" || jvm[1] != "-Xmx2G" {
		t.Fatalf("memory: %+v", jvm)
	}
	want := map[string]struct{}{}
	for _, arg := range mcmanifest.VelocityJVMFlags() {
		want[mcmanifest.JVMArgKey(arg)] = struct{}{}
	}
	for _, arg := range jvm[2:] {
		delete(want, mcmanifest.JVMArgKey(arg))
	}
	if len(want) != 0 {
		t.Fatalf("missing velocity flags: %+v jvm: %+v", want, jvm)
	}
	if countJVMArgKey(jvm, "-XX:G1HeapRegionSize") != 1 {
		t.Fatalf("region size: %+v", jvm)
	}
}

func TestGameServerJVMArgsSkipAikarForWaterfallAndBungee(t *testing.T) {
	t.Parallel()
	for _, serverType := range []string{"waterfall", "bungeecord"} {
		item := &models.GameServer{ServerType: serverType}
		jvm := gameServerJVMArgs(item)
		if !reflect.DeepEqual(jvm, []string{"-Xms2G", "-Xmx2G"}) {
			t.Fatalf("%s jvm: %+v", serverType, jvm)
		}
	}
}

func TestGameServerJVMArgsExtraOverridesAikar(t *testing.T) {
	t.Parallel()
	item := &models.GameServer{
		ServerType:   "paper",
		ExtraJVMArgs: models.StringList{"-XX:G1HeapRegionSize=16M"},
	}
	jvm := gameServerJVMArgs(item)
	if !hasAikarDefaults(jvm, "-Xms2G", "-Xmx2G") {
		t.Fatalf("jvm: %+v", jvm)
	}
	if countJVMArgKey(jvm, "-XX:G1HeapRegionSize") != 1 {
		t.Fatalf("region size dup: %+v", jvm)
	}
	found := false
	for _, arg := range jvm {
		if arg == "-XX:G1HeapRegionSize=16M" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("override missing: %+v", jvm)
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

func hasAikarDefaults(jvm []string, xms, xmx string) bool {
	if len(jvm) < 2 || jvm[0] != xms || jvm[1] != xmx {
		return false
	}
	want := map[string]struct{}{}
	for _, arg := range mcmanifest.AikarJVMFlags() {
		want[mcmanifest.JVMArgKey(arg)] = struct{}{}
	}
	for _, arg := range jvm[2:] {
		delete(want, mcmanifest.JVMArgKey(arg))
	}
	return len(want) == 0
}

func countJVMArgKey(jvm []string, key string) int {
	n := 0
	for _, arg := range jvm {
		if mcmanifest.JVMArgKey(arg) == key {
			n++
		}
	}
	return n
}
