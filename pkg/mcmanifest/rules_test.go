package mcmanifest

import (
	"encoding/json"
	"testing"
)

func TestRulesAllowDemoUserExcluded(t *testing.T) {
	raw := json.RawMessage(`{
		"rules": [{"action": "allow", "features": {"is_demo_user": true}}],
		"value": "--demo"
	}`)
	args := flattenArgumentList([]json.RawMessage{raw}, "windows")
	if len(args) != 0 {
		t.Fatalf("expected no demo arg, got %v", args)
	}
}

func TestRulesAllowCustomResolutionIncluded(t *testing.T) {
	raw := json.RawMessage(`{
		"rules": [{"action": "allow", "features": {"has_custom_resolution": true}}],
		"value": ["--width", "${resolution_width}"]
	}`)
	args := flattenArgumentList([]json.RawMessage{raw}, "windows")
	if len(args) != 2 || args[0] != "--width" {
		t.Fatalf("resolution args: %v", args)
	}
}
