package mcmanifest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	fabricLoaderVersionsURL = "https://meta.fabricmc.net/v2/versions/loader/%s"
	quiltLoaderVersionsURL  = "https://meta.quiltmc.org/v3/versions/loader/%s"
)

type profileLoaderEntry struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}

// ResolveLatestFabricLoaderVersion returns the newest stable Fabric loader for the given Minecraft version.
func (c *Client) ResolveLatestFabricLoaderVersion(ctx context.Context, mcVersion string) (string, error) {
	return c.resolveLatestProfileLoaderVersion(ctx, mcVersion, fabricLoaderVersionsURL, "fabric")
}

// ResolveLatestQuiltLoaderVersion returns the newest stable Quilt loader for the given Minecraft version.
func (c *Client) ResolveLatestQuiltLoaderVersion(ctx context.Context, mcVersion string) (string, error) {
	return c.resolveLatestProfileLoaderVersion(ctx, mcVersion, quiltLoaderVersionsURL, "quilt")
}

func (c *Client) resolveLatestProfileLoaderVersion(ctx context.Context, mcVersion, urlTemplate, label string) (string, error) {
	mcVersion = strings.TrimSpace(mcVersion)
	if mcVersion == "" {
		return "", fmt.Errorf("empty minecraft version")
	}
	body, err := c.get(ctx, fmt.Sprintf(urlTemplate, mcVersion))
	if err != nil {
		return "", fmt.Errorf("fetch %s loader versions: %w", label, err)
	}
	var entries []profileLoaderEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return "", fmt.Errorf("parse %s loader versions: %w", label, err)
	}
	latest := ""
	for _, entry := range entries {
		version := strings.TrimSpace(entry.Loader.Version)
		if version == "" || !isReleaseLoaderVersion(version) {
			continue
		}
		if latest == "" || compareSemanticVersions(version, latest) > 0 {
			latest = version
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no %s loader version found for minecraft %s", label, mcVersion)
	}
	return latest, nil
}

func compareSemanticVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(ap) {
			fmt.Sscanf(ap[i], "%d", &av)
		}
		if i < len(bp) {
			fmt.Sscanf(bp[i], "%d", &bv)
		}
		if av != bv {
			return av - bv
		}
	}
	return 0
}

func isReleaseLoaderVersion(version string) bool {
	lower := strings.ToLower(version)
	return !strings.Contains(lower, "beta") &&
		!strings.Contains(lower, "alpha") &&
		!strings.Contains(lower, "snapshot")
}
