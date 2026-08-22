package mcmanifest

import (
	"reflect"
	"testing"
)

func TestVelocityJVMFlags(t *testing.T) {
	t.Parallel()
	got := VelocityJVMFlags()
	want := []string{
		"-XX:+AlwaysPreTouch",
		"-XX:+ParallelRefProcEnabled",
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+UseG1GC",
		"-XX:G1HeapRegionSize=4M",
		"-XX:MaxInlineLevel=15",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flags: %+v", got)
	}
	got[0] = "mutated"
	if VelocityJVMFlags()[0] != "-XX:+AlwaysPreTouch" {
		t.Fatal("VelocityJVMFlags should return a copy")
	}
}

func TestAikarJVMFlags(t *testing.T) {
	t.Parallel()
	got := AikarJVMFlags()
	if len(got) != 20 {
		t.Fatalf("len: %d", len(got))
	}
	if got[0] != "-XX:+AlwaysPreTouch" || got[len(got)-1] != "-Daikars.new.flags=true" {
		t.Fatalf("flags: %+v", got)
	}
	got[0] = "mutated"
	if AikarJVMFlags()[0] != "-XX:+AlwaysPreTouch" {
		t.Fatal("AikarJVMFlags should return a copy")
	}
}

func TestMergeJVMArgsReplacesByKey(t *testing.T) {
	t.Parallel()
	got := MergeJVMArgs(
		[]string{"-Xms2G", "-Xmx2G", "-XX:+UseG1GC", "-XX:G1HeapRegionSize=8M"},
		[]string{"-XX:+UseG1GC", "-XX:G1HeapRegionSize=16M", "-Dfoo=bar"},
	)
	want := []string{"-Xms2G", "-Xmx2G", "-XX:+UseG1GC", "-XX:G1HeapRegionSize=16M", "-Dfoo=bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestJVMArgKey(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"-XX:+UseG1GC":            "-XX:UseG1GC",
		"-XX:-UseG1GC":            "-XX:UseG1GC",
		"-XX:G1HeapRegionSize=8M": "-XX:G1HeapRegionSize",
		"-Dusing.aikars.flags=https://mcflags.emc.gs": "-Dusing.aikars.flags",
		"-Xms2G": "-Xms",
		"-Xmx4G": "-Xmx",
	}
	for arg, want := range cases {
		if got := JVMArgKey(arg); got != want {
			t.Fatalf("%q: got %q want %q", arg, got, want)
		}
	}
}
