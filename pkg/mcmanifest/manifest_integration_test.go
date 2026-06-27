package mcmanifest

import (
	"context"
	"strings"
	"testing"
)

// TestIntegrationMojangManifest1_21 hits Mojang CDN (skipped with -short / CI).
func TestIntegrationMojangManifest1_21(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Mojang network test in short mode")
	}

	ctx := context.Background()
	client := NewClient()
	manifest, err := client.BuildInstanceManifest(ctx, "inst-e2e", "E2E", "1.21", "vanilla", "")
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if manifest.MainClass == "" {
		t.Fatal("expected main class")
	}
	if manifest.ClientJar.URL == "" {
		t.Fatal("expected client jar url")
	}
	if len(manifest.Libraries) == 0 {
		t.Fatal("expected libraries")
	}
	if manifest.JavaMajor < 17 {
		t.Fatalf("unexpected java major: %d", manifest.JavaMajor)
	}
	if !strings.HasPrefix(manifest.VersionURL, "https://") {
		t.Fatalf("version url: %q", manifest.VersionURL)
	}
}
