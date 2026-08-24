package mods

import (
	"path/filepath"
	"strings"
)

func keepVersionLoaderFilter(loader string) bool {
	return strings.EqualFold(strings.TrimSpace(loader), ProjectTypeDatapack)
}

func filterVersionsByLoader(items []Version, loader string) []Version {
	loader = strings.TrimSpace(loader)
	if loader == "" || len(items) == 0 {
		return items
	}
	out := make([]Version, 0, len(items))
	for _, item := range items {
		if versionMatchesLoader(item, loader) {
			out = append(out, item)
		}
	}
	return out
}

func versionMatchesLoader(item Version, loader string) bool {
	loader = strings.ToLower(strings.TrimSpace(loader))
	if loader == "" {
		return true
	}
	if loader == ProjectTypeDatapack {
		return versionIsDatapack(item)
	}
	if versionIsDatapackOnly(item) {
		return false
	}
	named := 0
	for _, candidate := range item.Loaders {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		named++
		if candidate == loader {
			return true
		}
	}
	return named == 0
}

func versionIsDatapack(item Version) bool {
	if hasLoaderName(item.Loaders, "datapack", "data pack") {
		return true
	}
	if hasModPlatformLoader(item.Loaders) {
		return false
	}
	return versionPrimaryExt(item) == ".zip"
}

func versionIsDatapackOnly(item Version) bool {
	return versionIsDatapack(item) && !hasModPlatformLoader(item.Loaders)
}

func hasLoaderName(loaders []string, names ...string) bool {
	for _, loader := range loaders {
		got := strings.ToLower(strings.TrimSpace(loader))
		for _, name := range names {
			if got == name {
				return true
			}
		}
	}
	return false
}

func hasModPlatformLoader(loaders []string) bool {
	for _, loader := range loaders {
		switch strings.ToLower(strings.TrimSpace(loader)) {
		case "fabric", "forge", "neoforge", "quilt", "bukkit", "spigot", "paper", "purpur", "folia",
			"velocity", "waterfall", "bungeecord", "sponge":
			return true
		}
	}
	return false
}

func versionPrimaryExt(item Version) string {
	if len(item.Files) == 0 {
		return ""
	}
	return strings.ToLower(filepath.Ext(item.Files[0].Filename))
}
