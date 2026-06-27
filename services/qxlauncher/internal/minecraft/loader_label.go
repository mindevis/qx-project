package minecraft

import (
	"fmt"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func LoaderDisplayLabel(loader string) string {
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case mcmanifest.LoaderNeoForge:
		return "NeoForge"
	case mcmanifest.LoaderForge:
		return "Forge"
	case mcmanifest.LoaderFabric:
		return "Fabric"
	case mcmanifest.LoaderQuilt:
		return "Quilt"
	case mcmanifest.LoaderVanilla, "":
		return "Vanilla"
	default:
		return loader
	}
}

func FormatLaunchLabel(manifest *mcmanifest.InstanceLaunchManifest, username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "Player"
	}
	if manifest == nil {
		return username
	}
	parts := []string{username}
	if manifest.MCVersion != "" {
		parts = append(parts, manifest.MCVersion)
	}
	loader := LoaderDisplayLabel(manifest.Loader)
	if loader != "Vanilla" {
		parts = append(parts, loader)
	}
	if manifest.LoaderVersion != "" && loader != "Vanilla" {
		parts = append(parts, manifest.LoaderVersion)
	}
	return strings.Join(parts, " · ")
}

func FormatLaunchLogFields(manifest *mcmanifest.InstanceLaunchManifest) []any {
	if manifest == nil {
		return nil
	}
	fields := []any{
		"loader", manifest.Loader,
		"mc_version", manifest.MCVersion,
	}
	if manifest.LoaderVersion != "" {
		fields = append(fields, "loader_version", manifest.LoaderVersion)
	}
	if manifest.VersionID != "" {
		fields = append(fields, "version_id", manifest.VersionID)
	}
	if manifest.LoaderClientJar.RelativePath != "" {
		fields = append(fields, "loader_client_jar", manifest.LoaderClientJar.RelativePath)
	}
	return fields
}

func FormatLaunchSummary(manifest *mcmanifest.InstanceLaunchManifest) string {
	if manifest == nil {
		return "Minecraft"
	}
	label := LoaderDisplayLabel(manifest.Loader)
	if label == "Vanilla" {
		return fmt.Sprintf("Minecraft %s", manifest.MCVersion)
	}
	if manifest.LoaderVersion != "" {
		return fmt.Sprintf("%s %s · MC %s", label, manifest.LoaderVersion, manifest.MCVersion)
	}
	return fmt.Sprintf("%s · MC %s", label, manifest.MCVersion)
}
