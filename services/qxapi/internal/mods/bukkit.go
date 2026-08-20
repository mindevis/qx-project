package mods

import "net/url"

func bukkitExternalURL(slug string) string {
	return "https://dev.bukkit.org/bukkit-plugins/" + url.PathEscape(slug)
}

func tagBukkitPlugins(items []SearchItem) []SearchItem {
	if len(items) == 0 {
		return items
	}
	out := make([]SearchItem, len(items))
	for i, item := range items {
		out[i] = tagBukkitPlugin(item)
	}
	return out
}

func tagBukkitPlugin(item SearchItem) SearchItem {
	item.Source = SourceBukkit
	item.ProjectType = ProjectTypePlugin
	if item.Slug != "" {
		item.ExternalURL = bukkitExternalURL(item.Slug)
	}
	return item
}

func tagBukkitProject(detail *ProjectDetail) *ProjectDetail {
	if detail == nil {
		return nil
	}
	out := *detail
	out.SearchItem = tagBukkitPlugin(detail.SearchItem)
	return &out
}
