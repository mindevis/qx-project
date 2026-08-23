package mods

import "testing"

func TestFilterVersionsByLoaderKeepsDatapackOnly(t *testing.T) {
	t.Parallel()
	datapack := Version{
		ID:      "dp",
		Loaders: []string{"datapack"},
		Files:   []VersionFile{{Filename: "Dungeons and Taverns v5.3.0.zip"}},
	}
	fabric := Version{
		ID:      "fab",
		Loaders: []string{"fabric"},
		Files:   []VersionFile{{Filename: "dungeons-and-taverns-5.3.0.jar"}},
	}
	forge := Version{
		ID:      "fg",
		Loaders: []string{"forge"},
		Files:   []VersionFile{{Filename: "dungeons-and-taverns-5.3.0.jar"}},
	}
	got := filterVersionsByLoader([]Version{fabric, datapack, forge}, "datapack")
	if len(got) != 1 || got[0].ID != "dp" {
		t.Fatalf("datapack filter: %+v", got)
	}
}

func TestFilterVersionsByLoaderDropsDatapackFromMods(t *testing.T) {
	t.Parallel()
	datapack := Version{ID: "dp", Loaders: []string{"datapack"}, Files: []VersionFile{{Filename: "pack.zip"}}}
	fabric := Version{ID: "fab", Loaders: []string{"fabric"}, Files: []VersionFile{{Filename: "mod.jar"}}}
	got := filterVersionsByLoader([]Version{datapack, fabric}, "fabric")
	if len(got) != 1 || got[0].ID != "fab" {
		t.Fatalf("fabric filter: %+v", got)
	}
}

func TestVersionIsDatapackFromZipWithoutLoader(t *testing.T) {
	t.Parallel()
	zipOnly := Version{ID: "zip", Files: []VersionFile{{Filename: "pack.zip"}}}
	if !versionIsDatapack(zipOnly) {
		t.Fatal("empty-loader zip should count as a datapack")
	}
	jarOnly := Version{ID: "jar", Files: []VersionFile{{Filename: "mod.jar"}}}
	if versionIsDatapack(jarOnly) {
		t.Fatal("empty-loader jar should not count as a datapack")
	}
}
