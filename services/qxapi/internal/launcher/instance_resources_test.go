package launcher

import (
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestRemoveInstanceResource(t *testing.T) {
	inst := &models.LauncherInstance{
		Mods: models.InstanceResourceList{
			{Source: "modrinth", ProjectID: "sodium", ProjectName: "Sodium", Filename: "sodium.jar", ResourceType: "mod"},
			{Source: "modrinth", ProjectID: "lithium", ProjectName: "Lithium", Filename: "lithium.jar", ResourceType: "mod"},
		},
	}

	if !removeInstanceResource(inst, DeleteInstanceResourceInput{
		Source: "modrinth", ProjectID: "sodium", Filename: "sodium.jar", ResourceType: "mod",
	}) {
		t.Fatal("expected removal to report true")
	}
	if len(inst.Mods) != 1 || inst.Mods[0].ProjectID != "lithium" {
		t.Fatalf("expected lithium only, got %+v", inst.Mods)
	}
}

func TestRemoveInstanceResourceNotFound(t *testing.T) {
	inst := &models.LauncherInstance{
		Mods: models.InstanceResourceList{
			{Source: "modrinth", ProjectID: "sodium", Filename: "sodium.jar", ResourceType: "mod"},
		},
	}

	if removeInstanceResource(inst, DeleteInstanceResourceInput{
		Source: "modrinth", ProjectID: "missing", ResourceType: "mod",
	}) {
		t.Fatal("expected removal to report false for a missing resource")
	}
	if len(inst.Mods) != 1 {
		t.Fatalf("list must be unchanged, got %+v", inst.Mods)
	}
}
