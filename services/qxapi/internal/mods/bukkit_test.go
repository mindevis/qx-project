package mods

import "testing"

func TestTagBukkitPluginUsesBukkitURL(t *testing.T) {
	t.Parallel()
	item := tagBukkitPlugin(SearchItem{
		ID:          "123",
		Source:      SourceCurseForge,
		Slug:        "protocollib",
		Name:        "ProtocolLib",
		ProjectType: ProjectTypePlugin,
		ExternalURL: "https://www.curseforge.com/minecraft/bukkit-plugins/protocollib",
	})
	if item.Source != SourceBukkit {
		t.Fatalf("source: %s", item.Source)
	}
	if item.ExternalURL != "https://dev.bukkit.org/bukkit-plugins/protocollib" {
		t.Fatalf("url: %s", item.ExternalURL)
	}
}
