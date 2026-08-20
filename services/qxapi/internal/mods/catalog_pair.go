package mods

import (
	"regexp"
	"sort"
	"strings"
)

var catalogNameDecoration = regexp.MustCompile(`\s*[\(\[][^\)\]]*[\)\]]`)

func catalogNameKey(name string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func stripCatalogNameDecorations(name string) string {
	stripped := catalogNameDecoration.ReplaceAllString(name, " ")
	return strings.TrimSpace(stripped)
}

func catalogMatchKeys(item SearchItem) []string {
	keys := make([]string, 0, 3)
	if slug := strings.ToLower(strings.TrimSpace(item.Slug)); slug != "" {
		keys = append(keys, "s:"+slug)
	}
	name := catalogNameKey(item.Name)
	if name != "" {
		keys = append(keys, "n:"+name)
	}
	if stripped := catalogNameKey(stripCatalogNameDecorations(item.Name)); stripped != "" && stripped != name {
		keys = append(keys, "n:"+stripped)
	}
	return keys
}

func indexCatalogKeys(items []SearchItem) map[string][]int {
	index := make(map[string][]int)
	for i, item := range items {
		for _, key := range catalogMatchKeys(item) {
			index[key] = append(index[key], i)
		}
	}
	return index
}

func findCatalogPartner(item SearchItem, pool []SearchItem, used []bool, index map[string][]int) int {
	for _, key := range catalogMatchKeys(item) {
		for _, idx := range index[key] {
			if used[idx] {
				continue
			}
			if pool[idx].Source == item.Source {
				continue
			}
			return idx
		}
	}
	return -1
}

// pairAndInterleaveSearch keeps mixed-catalog partners on the same page so the
// UI can show one card with Modrinth and CurseForge. limit is the number of
// cards, not raw listings.
func pairAndInterleaveSearch(primary, secondary []SearchItem, limit int) []SearchItem {
	limit = clampSearchLimit(limit)
	usedP := make([]bool, len(primary))
	usedS := make([]bool, len(secondary))
	pIndex := indexCatalogKeys(primary)
	sIndex := indexCatalogKeys(secondary)

	out := make([]SearchItem, 0, min(limit*2, maxSearchItems*2))
	cards := 0
	i, j := 0, 0
	for cards < limit && (i < len(primary) || j < len(secondary)) {
		for i < len(primary) && usedP[i] {
			i++
		}
		if i < len(primary) && cards < limit {
			item := primary[i]
			usedP[i] = true
			i++
			out = append(out, item)
			if p := findCatalogPartner(item, secondary, usedS, sIndex); p >= 0 {
				usedS[p] = true
				out = append(out, secondary[p])
			}
			cards++
		}
		for j < len(secondary) && usedS[j] {
			j++
		}
		if j < len(secondary) && cards < limit {
			item := secondary[j]
			usedS[j] = true
			j++
			out = append(out, item)
			if p := findCatalogPartner(item, primary, usedP, pIndex); p >= 0 {
				usedP[p] = true
				out = append(out, primary[p])
			}
			cards++
		}
	}
	return out
}

func preferQueryMatches(items []SearchItem, query string) []SearchItem {
	q := catalogNameKey(query)
	if q == "" || len(items) < 2 {
		return items
	}
	score := func(item SearchItem) int {
		name := catalogNameKey(item.Name)
		slug := strings.ToLower(strings.TrimSpace(item.Slug))
		switch {
		case name == q || slug == q:
			return 3
		case strings.HasPrefix(name, q) || strings.HasPrefix(slug, q):
			return 2
		case strings.Contains(name, q) || strings.Contains(slug, q):
			return 1
		default:
			return 0
		}
	}
	out := append([]SearchItem(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		return score(out[i]) > score(out[j])
	})
	return out
}
