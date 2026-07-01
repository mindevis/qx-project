package mcmanifest

import "testing"

func TestApplyExtraJVMArgsAppends(t *testing.T) {
	manifest := &InstanceLaunchManifest{
		JVMArguments: []string{"-Xms1G", "-Xmx2G"},
	}
	ApplyExtraJVMArgs(manifest, []string{"-XX:+UseG1GC", "-Dfoo=bar"})
	if len(manifest.JVMArguments) != 4 {
		t.Fatalf("expected 4 args, got %v", manifest.JVMArguments)
	}
	if manifest.JVMArguments[2] != "-XX:+UseG1GC" || manifest.JVMArguments[3] != "-Dfoo=bar" {
		t.Fatalf("unexpected args: %v", manifest.JVMArguments)
	}
}

func TestApplyWindowSizeAppends(t *testing.T) {
	width, height := 1280, 720
	manifest := &InstanceLaunchManifest{
		GameArguments: []string{"--version", "1.21"},
	}
	ApplyWindowSize(manifest, &width, &height)
	want := []string{"--version", "1.21", "--width", "1280", "--height", "720"}
	if len(manifest.GameArguments) != len(want) {
		t.Fatalf("got %v", manifest.GameArguments)
	}
	for i := range want {
		if manifest.GameArguments[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, manifest.GameArguments[i], want[i])
		}
	}
}

func TestApplyWindowSizeReplacesExisting(t *testing.T) {
	width, height := 1920, 1080
	manifest := &InstanceLaunchManifest{
		GameArguments: []string{"--width", "800", "--height", "600"},
	}
	ApplyWindowSize(manifest, &width, &height)
	if manifest.GameArguments[1] != "1920" || manifest.GameArguments[3] != "1080" {
		t.Fatalf("unexpected args: %v", manifest.GameArguments)
	}
}

func TestSanitizeJVMArgs(t *testing.T) {
	got := SanitizeJVMArgs([]string{"  -Xmx2G  ", "", "  ", "-Dfoo=bar"})
	want := []string{"-Xmx2G", "-Dfoo=bar"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}
