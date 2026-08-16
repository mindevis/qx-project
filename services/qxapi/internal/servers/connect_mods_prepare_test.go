package servers

import "testing"

func TestIsPullableResourceFilename(t *testing.T) {
	if !isPullableResourceFilename("Mowzie's Mobs-1.20.1.jar") {
		t.Fatal("expected jar with apostrophe to be pullable")
	}
	if isPullableResourceFilename("readme.txt") {
		t.Fatal("expected txt to be skipped")
	}
}

func TestAgentPayloadError(t *testing.T) {
	if err := agentPayloadError([]byte(`{"error":"content too large"}`)); err == nil || err.Error() != "content too large" {
		t.Fatalf("expected content too large, got %v", err)
	}
	if err := agentPayloadError([]byte(`{"status":"ok","content_b64":"abc"}`)); err != nil {
		t.Fatalf("expected no error for success payload, got %v", err)
	}
}

func TestEnabledClientModSet(t *testing.T) {
	set := enabledClientModSet([]string{"JourneyMap.jar", "journeymap.jar", "  ", "AppleSkin.jar"})
	if !set["journeymap.jar"] || !set["appleskin.jar"] {
		t.Fatalf("unexpected set: %#v", set)
	}
	if len(set) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(set))
	}
}
