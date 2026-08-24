package mods

import "testing"

func TestNormalizeAndInferVersionType(t *testing.T) {
	if got := NormalizeVersionType("Alpha"); got != VersionTypeAlpha {
		t.Fatalf("alpha: %q", got)
	}
	if got := NormalizeVersionType("SNAPSHOT"); got != VersionTypeBeta {
		t.Fatalf("snapshot: %q", got)
	}
	if got := NormalizeVersionType("stable"); got != VersionTypeRelease {
		t.Fatalf("stable: %q", got)
	}
	if got := InferVersionType("sodium-0.6.0+mc1.21.1.jar"); got != VersionTypeRelease {
		t.Fatalf("release jar: %q", got)
	}
	if got := InferVersionType("mod-1.0.0-alpha.3.jar"); got != VersionTypeAlpha {
		t.Fatalf("alpha jar: %q", got)
	}
	if got := InferVersionType("plugin-2.0.0-rc.1.jar"); got != VersionTypeBeta {
		t.Fatalf("rc jar: %q", got)
	}
	if got := CurseForgeReleaseType(3); got != VersionTypeAlpha {
		t.Fatalf("cf alpha: %q", got)
	}
	if got := CurseForgeReleaseType(2); got != VersionTypeBeta {
		t.Fatalf("cf beta: %q", got)
	}
	if got := CurseForgeReleaseType(1); got != VersionTypeRelease {
		t.Fatalf("cf release: %q", got)
	}
}
