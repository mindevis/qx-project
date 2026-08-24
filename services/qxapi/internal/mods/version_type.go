package mods

import "strings"

const (
	VersionTypeRelease = "release"
	VersionTypeBeta    = "beta"
	VersionTypeAlpha   = "alpha"
)

// NormalizeVersionType maps catalog/provider labels onto release, beta, or alpha.
func NormalizeVersionType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case VersionTypeAlpha, "a":
		return VersionTypeAlpha
	case VersionTypeBeta, "b", "rc", "pre", "prerelease", "preview", "snapshot":
		return VersionTypeBeta
	case VersionTypeRelease, "stable", "ga", "final":
		return VersionTypeRelease
	default:
		return ""
	}
}

// InferVersionType guesses a channel from a version or file name when the
// provider did not send an explicit type.
func InferVersionType(names ...string) string {
	for _, name := range names {
		if typ := NormalizeVersionType(name); typ != "" {
			return typ
		}
		lower := strings.ToLower(name)
		switch {
		case strings.Contains(lower, "alpha"):
			return VersionTypeAlpha
		case strings.Contains(lower, "beta"),
			strings.Contains(lower, "snapshot"),
			strings.Contains(lower, "preview"),
			strings.Contains(lower, "-pre"),
			strings.Contains(lower, "-rc"):
			return VersionTypeBeta
		}
	}
	return VersionTypeRelease
}

// CurseForgeReleaseType maps CurseForge file.releaseType: 1 release, 2 beta, 3 alpha.
func CurseForgeReleaseType(code int) string {
	switch code {
	case 3:
		return VersionTypeAlpha
	case 2:
		return VersionTypeBeta
	case 1:
		return VersionTypeRelease
	default:
		return ""
	}
}
