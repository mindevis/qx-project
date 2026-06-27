package mcmanifest

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

const neoForgeMavenMetadataURL = neoforgeMavenBase + "/maven-metadata.xml"

type mavenMetadata struct {
	Versioning struct {
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
}

// NeoForgeVersionPrefix returns the NeoForge Maven version prefix for a Minecraft version (e.g. 1.21.1 → 21.1.).
func NeoForgeVersionPrefix(mcVersion string) string {
	parts := strings.Split(strings.TrimSpace(mcVersion), ".")
	if len(parts) < 2 || parts[0] != "1" {
		return ""
	}
	switch len(parts) {
	case 2:
		return parts[1] + ".0."
	case 3:
		return parts[1] + "." + parts[2] + "."
	default:
		return ""
	}
}

// NeoForgeMcVersion maps a NeoForge loader version to its Minecraft version (e.g. 21.1.234 → 1.21.1).
func NeoForgeMcVersion(loaderVersion string) string {
	parts := strings.Split(strings.TrimSpace(loaderVersion), ".")
	if len(parts) < 2 {
		return ""
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	switch major {
	case 21:
		if parts[1] == "0" {
			return "1.21"
		}
		return "1.21." + parts[1]
	case 20:
		return "1.20." + parts[1]
	default:
		return ""
	}
}

func compareNeoForgeVersions(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	n := len(ap)
	if len(bp) > n {
		n = len(bp)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(ap) {
			av, _ = strconv.Atoi(ap[i])
		}
		if i < len(bp) {
			bv, _ = strconv.Atoi(bp[i])
		}
		if av != bv {
			return av - bv
		}
	}
	return 0
}

// ResolveLatestNeoForgeVersion returns the newest NeoForge release for the given Minecraft version.
func (c *Client) ResolveLatestNeoForgeVersion(ctx context.Context, mcVersion string) (string, error) {
	mcVersion = strings.TrimSpace(mcVersion)
	if mcVersion == "" {
		return "", fmt.Errorf("empty minecraft version")
	}
	body, err := c.get(ctx, neoForgeMavenMetadataURL)
	if err != nil {
		return "", fmt.Errorf("fetch neoforge metadata: %w", err)
	}
	var meta mavenMetadata
	if err := xml.Unmarshal(body, &meta); err != nil {
		return "", fmt.Errorf("parse neoforge metadata: %w", err)
	}
	latest := ""
	prefix := NeoForgeVersionPrefix(mcVersion)
	if prefix == "" {
		return "", fmt.Errorf("unsupported minecraft version %q for neoforge", mcVersion)
	}
	for _, version := range meta.Versioning.Versions.Version {
		if strings.Contains(version, "beta") || strings.Contains(version, "alpha") {
			continue
		}
		if !strings.HasPrefix(version, prefix) {
			continue
		}
		if latest == "" || compareNeoForgeVersions(version, latest) > 0 {
			latest = version
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no neoforge version found for minecraft %s", mcVersion)
	}
	return latest, nil
}
