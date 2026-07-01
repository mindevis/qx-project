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

func TestApplyMinMemoryMBReplacesExisting(t *testing.T) {
	manifest := &InstanceLaunchManifest{
		JVMArguments: []string{"-Xms1G", "-Xmx2G"},
	}
	ApplyMinMemoryMB(manifest, 2048)
	if manifest.JVMArguments[0] != "-Xms2G" {
		t.Fatalf("expected -Xms2G, got %q", manifest.JVMArguments[0])
	}
}

func TestApplyMinMemoryMBPrependsWhenMissing(t *testing.T) {
	manifest := &InstanceLaunchManifest{
		JVMArguments: []string{"-XX:+UseG1GC"},
	}
	ApplyMinMemoryMB(manifest, 1024)
	if manifest.JVMArguments[0] != "-Xms1G" {
		t.Fatalf("expected -Xms1G, got %q", manifest.JVMArguments[0])
	}
}

func TestApplyMinMemoryMBNonGigabyte(t *testing.T) {
	manifest := &InstanceLaunchManifest{}
	ApplyMinMemoryMB(manifest, 1536)
	if manifest.JVMArguments[0] != "-Xms1536M" {
		t.Fatalf("expected -Xms1536M, got %q", manifest.JVMArguments[0])
	}
}
