package cosmetics

import "strings"

// CatalogEntry is a browsable skin from a public Minecraft profile (Mojang).
type CatalogEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Username   string `json:"username"`
	Category   string `json:"category"`
	PreviewURL string `json:"preview_url"`
}

var skinCatalog = []CatalogEntry{
	{ID: "dream", Name: "Dream", Username: "Dream", Category: "popular"},
	{ID: "technoblade", Name: "Technoblade", Username: "Technoblade", Category: "popular"},
	{ID: "philza", Name: "Philza", Username: "Philza", Category: "popular"},
	{ID: "tommyinnit", Name: "TommyInnit", Username: "TommyInnit", Category: "popular"},
	{ID: "grian", Name: "Grian", Username: "Grian", Category: "creators"},
	{ID: "mumbojumbo", Name: "Mumbo Jumbo", Username: "MumboJumbo", Category: "creators"},
	{ID: "captainsparklez", Name: "CaptainSparklez", Username: "CaptainSparklez", Category: "creators"},
	{ID: "notch", Name: "Notch", Username: "Notch", Category: "classic"},
	{ID: "jeb_", Name: "jeb_", Username: "jeb_", Category: "classic"},
	{ID: "herobrine", Name: "Herobrine", Username: "Herobrine", Category: "classic"},
	{ID: "ranboo", Name: "Ranboo", Username: "Ranboo", Category: "popular"},
	{ID: "sapnap", Name: "Sapnap", Username: "Sapnap", Category: "popular"},
}

func init() {
	for i := range skinCatalog {
		skinCatalog[i].Source = "mojang"
		skinCatalog[i].PreviewURL = "https://mc-heads.net/avatar/" + skinCatalog[i].Username + "/80"
	}
}

// ListSkinCatalog returns curated skins, optionally filtered by category.
func ListSkinCatalog(category string) []CatalogEntry {
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		out := make([]CatalogEntry, len(skinCatalog))
		copy(out, skinCatalog)
		return out
	}
	var out []CatalogEntry
	for _, entry := range skinCatalog {
		if entry.Category == category {
			out = append(out, entry)
		}
	}
	return out
}

// CatalogEntryByID finds a catalog entry by id.
func CatalogEntryByID(id string) (CatalogEntry, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, entry := range skinCatalog {
		if entry.ID == id {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}
