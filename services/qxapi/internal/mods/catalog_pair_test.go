package mods

import "testing"

func TestPairAndInterleaveSearchAttachesPartnerFromLaterInList(t *testing.T) {
	t.Parallel()
	cf := []SearchItem{
		{ID: "cf-cloth", Source: SourceCurseForge, Slug: "cloth-config", Name: "Cloth Config API"},
		{ID: "cf-minimap", Source: SourceCurseForge, Slug: "xaeros-minimap", Name: "Xaero's Minimap"},
	}
	mr := []SearchItem{
		{ID: "mr-minimap", Source: SourceModrinth, Slug: "xaeros-minimap", Name: "Xaero's Minimap"},
		{ID: "mr-lithium", Source: SourceModrinth, Slug: "lithium", Name: "Lithium"},
	}

	out := pairAndInterleaveSearch(cf, mr, 4)
	if len(out) < 4 {
		t.Fatalf("expected paired listings, got %+v", out)
	}
	var sawMinimapPair bool
	for i := 0; i < len(out)-1; i++ {
		if out[i].Slug == "xaeros-minimap" && out[i+1].Slug == "xaeros-minimap" && out[i].Source != out[i+1].Source {
			sawMinimapPair = true
			break
		}
	}
	if !sawMinimapPair {
		t.Fatalf("expected minimap listings to sit next to each other, got %+v", out)
	}
}

func TestPairAndInterleaveSearchMatchesSlugAndStrippedName(t *testing.T) {
	t.Parallel()
	cf := []SearchItem{{ID: "238222", Source: SourceCurseForge, Slug: "jei", Name: "Just Enough Items (JEI)"}}
	mr := []SearchItem{{ID: "jei", Source: SourceModrinth, Slug: "jei", Name: "JEI"}}
	out := pairAndInterleaveSearch(cf, mr, 1)
	if len(out) != 2 {
		t.Fatalf("expected one paired card, got %+v", out)
	}
}

func TestPreferQueryMatchesRanksExactName(t *testing.T) {
	t.Parallel()
	out := preferQueryMatches([]SearchItem{
		{Name: "FurnitureLib", Slug: "furniturelib", Source: SourceModrinth},
		{Name: "ProtocolLib", Slug: "ProtocolLib", Source: SourceHangar},
	}, "ProtocolLib")
	if out[0].Name != "ProtocolLib" {
		t.Fatalf("expected ProtocolLib first, got %+v", out)
	}
}

func TestCatalogNameKey(t *testing.T) {
	t.Parallel()
	if catalogNameKey("  Xaero's Minimap ") != "xaero's minimap" {
		t.Fatalf("name key: %q", catalogNameKey("  Xaero's Minimap "))
	}
	if stripCatalogNameDecorations("YetAnotherConfigLib (YACL)") != "YetAnotherConfigLib" {
		t.Fatalf("strip: %q", stripCatalogNameDecorations("YetAnotherConfigLib (YACL)"))
	}
}
