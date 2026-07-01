package cosmetics

import "testing"

func TestListSkinCatalog(t *testing.T) {
	all := ListSkinCatalog("")
	if len(all) < 10 {
		t.Fatalf("expected catalog entries, got %d", len(all))
	}
	popular := ListSkinCatalog("popular")
	if len(popular) == 0 {
		t.Fatal("expected popular category")
	}
	for _, entry := range popular {
		if entry.Category != "popular" {
			t.Fatalf("category: %q", entry.Category)
		}
	}
	if _, ok := CatalogEntryByID("dream"); !ok {
		t.Fatal("dream entry missing")
	}
}
