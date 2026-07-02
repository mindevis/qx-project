package servers

import "testing"

func TestEnabledClientModSet(t *testing.T) {
	set := enabledClientModSet([]string{"JourneyMap.jar", "journeymap.jar", "  ", "AppleSkin.jar"})
	if !set["journeymap.jar"] || !set["appleskin.jar"] {
		t.Fatalf("unexpected set: %#v", set)
	}
	if len(set) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(set))
	}
}
