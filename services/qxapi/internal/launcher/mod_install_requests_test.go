package launcher

import (
	"path/filepath"
	"testing"
)

func TestResourceInstallFolder(t *testing.T) {
	tests := map[string]string{
		"mod":          "mods",
		"resourcepack": "resourcepacks",
		"shader":       "shaderpacks",
		"datapack":     filepath.Join("saves", "world", "datapacks"),
	}
	for resourceType, want := range tests {
		if got := ResourceInstallFolder(resourceType); got != want {
			t.Fatalf("ResourceInstallFolder(%q) = %q, want %q", resourceType, got, want)
		}
	}
}

func TestNormalizeResourceTypeDatapack(t *testing.T) {
	if got := normalizeResourceType("datapack"); got != "datapack" {
		t.Fatalf("normalizeResourceType(datapack) = %q", got)
	}
}
