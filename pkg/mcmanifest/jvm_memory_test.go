package mcmanifest

import "testing"

func TestApplyMaxMemoryMBReplacesExisting(t *testing.T) {
	manifest := &InstanceLaunchManifest{
		JVMArguments: []string{"-Xmx2G", "-XX:+UnlockExperimentalVMOptions"},
	}
	ApplyMaxMemoryMB(manifest, 4096)
	if manifest.JVMArguments[0] != "-Xmx4G" {
		t.Fatalf("expected -Xmx4G, got %q", manifest.JVMArguments[0])
	}
}

func TestApplyMaxMemoryMBPrependsWhenMissing(t *testing.T) {
	manifest := &InstanceLaunchManifest{
		JVMArguments: []string{"-XX:+UseG1GC"},
	}
	ApplyMaxMemoryMB(manifest, 3072)
	if manifest.JVMArguments[0] != "-Xmx3G" {
		t.Fatalf("expected -Xmx3G, got %q", manifest.JVMArguments[0])
	}
}

func TestApplyMaxMemoryMBNonGigabyte(t *testing.T) {
	manifest := &InstanceLaunchManifest{}
	ApplyMaxMemoryMB(manifest, 1536)
	if manifest.JVMArguments[0] != "-Xmx1536M" {
		t.Fatalf("expected -Xmx1536M, got %q", manifest.JVMArguments[0])
	}
}
