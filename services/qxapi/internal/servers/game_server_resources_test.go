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

func TestAppendUniqueContentResourceReplacesDifferentFilename(t *testing.T) {
	list := appendUniqueContentResource(nil, models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-0.5.0.jar", ResourceType: "mod", SideOverride: "server",
	})
	list = appendUniqueContentResource(list, models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-0.6.0.jar", ResourceType: "mod", SideOverride: "server", VersionNumber: "0.6.0",
	})
	if len(list) != 1 || list[0].Filename != "sodium-0.6.0.jar" || list[0].VersionNumber != "0.6.0" {
		t.Fatalf("got %+v", list)
	}
}

func TestAppendUniqueContentResourceKeepsClientAndServerCopies(t *testing.T) {
	list := appendUniqueContentResource(nil, models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-server.jar", ResourceType: "mod", SideOverride: "server",
	})
	list = appendUniqueContentResource(list, models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-client.jar", ResourceType: "mod", SideOverride: "client",
	})
	if len(list) != 2 {
		t.Fatalf("got %+v", list)
	}
}

func TestShouldReplaceInstalledContent(t *testing.T) {
	existing := &models.InstanceResourceEntry{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium-0.5.0.jar", VersionID: "v1",
	}
	if !ShouldReplaceInstalledContent(existing, "sodium-0.5.0.jar", "sodium-0.6.0.jar", "v2") {
		t.Fatal("changing version with a new filename must replace")
	}
	if ShouldReplaceInstalledContent(existing, "", "sodium-0.5.0.jar", "v1") {
		t.Fatal("same version and filename must not replace")
	}
	if !ShouldReplaceInstalledContent(existing, "sodium-0.5.0.jar.disabled", "sodium-0.6.0.jar", "v2") {
		t.Fatal("disabled old file must still be replaced")
	}
	if ShouldReplaceInstalledContent(nil, "", "fresh.jar", "v1") {
		t.Fatal("first install must not look like a replace")
	}
}

func TestContentFilesToReplace(t *testing.T) {
	existing := &models.InstanceResourceEntry{Filename: "sodium-0.5.0.jar"}
	got := ContentFilesToReplace(existing, "sodium-0.5.0.jar.disabled", "sodium-0.6.0.jar")
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	same := ContentFilesToReplace(existing, "sodium-0.5.0.jar", "sodium-0.5.0.jar")
	if len(same) != 0 {
		t.Fatalf("same filename should not delete, got %v", same)
	}
}

func TestFindContentResourceByProject(t *testing.T) {
	list := models.InstanceResourceList{
		{Source: "modrinth", ProjectID: "sodium", Filename: "sodium-server.jar", SideOverride: "server"},
		{Source: "modrinth", ProjectID: "sodium", Filename: "sodium-client.jar", SideOverride: "client"},
	}
	server := findContentResourceByProject(list, "modrinth", "sodium", "server")
	if server == nil || server.Filename != "sodium-server.jar" {
		t.Fatalf("server=%+v", server)
	}
	client := findContentResourceByProject(list, "modrinth", "sodium", "client")
	if client == nil || client.Filename != "sodium-client.jar" {
		t.Fatalf("client=%+v", client)
	}
}

func TestRemoveContentResource(t *testing.T) {
	list := models.InstanceResourceList{
		{Filename: "keep.jar", ResourceType: "mod", SideOverride: "server"},
		{Filename: "gone.jar", ResourceType: "mod", SideOverride: "server"},
		{Filename: "client.jar", ResourceType: "mod", SideOverride: "client"},
		{Filename: "shared.jar", ResourceType: "mod", SideOverride: "both"},
	}
	got := removeContentResource(list, "gone.jar", "mod", "server")
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	got = removeContentResource(got, "client.jar", "mod", "client")
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	got = removeContentResource(got, "shared.jar", "mod", "server")
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
	both := resourceEntryFromSync("mod", mods.SyncModRequest{
		Filename:     "jei.jar",
		ModTarget:    "",
		SideOverride: "both",
	})
	if both.SideOverride != "both" {
		t.Fatalf("both=%+v", both)
	}
}

func TestMatchesContentTargetFilterIncludesBothOnServerFolder(t *testing.T) {
	if !matchesContentTargetFilter("both", "server") {
		t.Fatal("both-sided mods live in mods/")
	}
	if matchesContentTargetFilter("client", "server") {
		t.Fatal("client mods must not match the server folder filter")
	}
	if !matchesContentTargetFilter("client", "client") {
		t.Fatal("client filter should keep client mods")
	}
}

func TestContentSideForFilenameAndPull(t *testing.T) {
	list := models.InstanceResourceList{
		{Filename: "map.jar", SideOverride: "server"},
		{Filename: "jei.jar", SideOverride: "both"},
		{Filename: "legacy.jar"},
	}
	if contentSideForFilename(list, "map.jar") != "server" {
		t.Fatal("expected server")
	}
	if shouldPullServerModToClient(contentSideForFilename(list, "map.jar")) {
		t.Fatal("server-only mods must stay off the client")
	}
	if !shouldPullServerModToClient(contentSideForFilename(list, "jei.jar")) {
		t.Fatal("both-sided mods must reach the client")
	}
	if !shouldPullServerModToClient(contentSideForFilename(list, "legacy.jar")) {
		t.Fatal("legacy mods without a side stay on the client")
	}
	if !shouldPullServerModToClient(contentSideForFilename(list, "unknown.jar")) {
		t.Fatal("unknown mods/ files keep the previous pull behavior")
	}
}

func TestContentModTargetForSide(t *testing.T) {
	if contentModTargetForSide("mod", "client") != "client-mods" {
		t.Fatal("client mods belong in client-mods/")
	}
	if contentModTargetForSide("mod", "both") != "" {
		t.Fatal("both-sided mods belong in the default mods/ folder")
	}
	if contentModTargetForSide("mod", "server") != "" {
		t.Fatal("server-only mods belong in the default mods/ folder")
	}
}
