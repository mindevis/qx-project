package launcher

import (
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestFindUploadedInstanceResource(t *testing.T) {
	inst := &models.LauncherInstance{
		Mods: models.InstanceResourceList{
			{
				Source:       "upload",
				ProjectName:  "Custom",
				Filename:     "custom.jar",
				ResourceType: "mod",
			},
			{
				Source:       "modrinth",
				ProjectID:    "sodium",
				ProjectName:  "Sodium",
				Filename:     "sodium.jar",
				ResourceType: "mod",
			},
		},
	}

	if got := FindUploadedInstanceResource(inst, "custom.jar", "mod"); got == nil {
		t.Fatal("expected uploaded mod")
	}
	if got := FindUploadedInstanceResource(inst, "sodium.jar", "mod"); got != nil {
		t.Fatal("catalog mod should not match upload lookup")
	}
	if got := FindUploadedInstanceResource(inst, "missing.jar", "mod"); got != nil {
		t.Fatal("expected nil for missing file")
	}
}
