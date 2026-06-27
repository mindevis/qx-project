package mcmanifest

import (
	"encoding/json"
	"testing"
)

func TestMergeVersionArguments(t *testing.T) {
	parent := &VersionArguments{
		Game: []json.RawMessage{
			json.RawMessage(`"--username"`),
			json.RawMessage(`"${auth_player_name}"`),
		},
		JVM: []json.RawMessage{json.RawMessage(`"-Xmx2G"`)},
	}
	child := &VersionArguments{
		Game: []json.RawMessage{json.RawMessage(`"--launchTarget"`), json.RawMessage(`"forgeclient"`)},
		JVM: []json.RawMessage{
			json.RawMessage(`"-p"`),
			json.RawMessage(`"mods.jar"`),
		},
	}
	merged := mergeVersionArguments(parent, child)
	if len(merged.Game) != 4 || len(merged.JVM) != 2 {
		t.Fatalf("merged: %+v", merged)
	}
}
