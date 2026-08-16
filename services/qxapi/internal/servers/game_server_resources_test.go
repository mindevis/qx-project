package servers

import (
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/mods"
)

func TestAppendUniqueContentResourceReplacesSameProject(t *testing.T) {
	list := appendUniqueContentResource(nil, models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-1.jar", ResourceType: "mod",
	})
	list = appendUniqueContentResource(list, models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-1.jar", ResourceType: "mod", VersionNumber: "2",
	})
	if len(list) != 1 || list[0].VersionNumber != "2" {
		t.Fatalf("got %+v", list)
	}
}

func TestRemoveContentResource(t *testing.T) {
	list := models.InstanceResourceList{
		{Filename: "keep.jar", ResourceType: "mod", SideOverride: "server"},
		{Filename: "gone.jar", ResourceType: "mod", SideOverride: "server"},
		{Filename: "client.jar", ResourceType: "mod", SideOverride: "client"},
	}
	got := removeContentResource(list, "gone.jar", "mod", "server")
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	got = removeContentResource(got, "client.jar", "mod", "client")
	if len(got) != 1 || got[0].Filename != "keep.jar" {
		t.Fatalf("got %+v", got)
	}
}

func TestResourceEntryFromSync(t *testing.T) {
	entry := resourceEntryFromSync("plugin", mods.SyncModRequest{
		Source:      "modrinth",
		ProjectID:   "essentials",
		Filename:    "EssentialsX.jar",
		ProjectName: "EssentialsX",
		ModTarget:   "",
	})
	if entry.ResourceType != "plugin" || entry.SideOverride != "server" || entry.ProjectName != "EssentialsX" {
		t.Fatalf("got %+v", entry)
	}
	client := resourceEntryFromSync("mod", mods.SyncModRequest{
		Filename:  "map.jar",
		ModTarget: "client-mods",
	})
	if client.SideOverride != "client" || client.ProjectName != "map.jar" {
		t.Fatalf("client=%+v", client)
	}
}
