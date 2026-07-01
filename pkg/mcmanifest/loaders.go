package mcmanifest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	LoaderVanilla  = "vanilla"
	LoaderForge    = "forge"
	LoaderNeoForge = "neoforge"
	LoaderFabric   = "fabric"
	LoaderQuilt    = "quilt"
)

func NormalizeLoader(loader string) string {
	loader = strings.ToLower(strings.TrimSpace(loader))
	if loader == "" {
		return LoaderVanilla
	}
	return loader
}

func IsSupportedLoader(loader string) bool {
	switch NormalizeLoader(loader) {
	case LoaderVanilla, LoaderForge, LoaderNeoForge, LoaderFabric, LoaderQuilt:
		return true
	default:
		return false
	}
}

func LoaderRequiresVersion(loader string) bool {
	return NormalizeLoader(loader) != LoaderVanilla
}

type VersionArguments struct {
	Game []json.RawMessage `json:"game"`
	JVM  []json.RawMessage `json:"jvm"`
}

func (c *Client) BuildInstanceManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion, targetOS string) (*InstanceLaunchManifest, error) {
	loader = NormalizeLoader(loader)
	loaderVersion = strings.TrimSpace(loaderVersion)
	targetOS = NormalizeTargetOS(targetOS)
	if !IsSupportedLoader(loader) {
		return nil, fmt.Errorf("unsupported loader %q", loader)
	}
	if LoaderRequiresVersion(loader) && loaderVersion == "" {
		return nil, fmt.Errorf("loader version required for %s", loader)
	}

	switch loader {
	case LoaderVanilla:
		return c.buildVanillaManifest(ctx, instanceID, name, mcVersion, loader, targetOS)
	case LoaderFabric:
		url := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s/%s/profile/json", mcVersion, loaderVersion)
		return c.buildProfileManifest(ctx, instanceID, name, mcVersion, loader, loaderVersion, url, targetOS)
	case LoaderQuilt:
		url := fmt.Sprintf("https://meta.quiltmc.org/v3/versions/loader/%s/%s/profile/json", mcVersion, loaderVersion)
		return c.buildProfileManifest(ctx, instanceID, name, mcVersion, loader, loaderVersion, url, targetOS)
	case LoaderForge:
		return c.buildInstallerManifest(ctx, instanceID, name, mcVersion, loader, loaderVersion, forgeInstallerURL(mcVersion, loaderVersion), targetOS)
	case LoaderNeoForge:
		return c.buildInstallerManifest(ctx, instanceID, name, mcVersion, loader, loaderVersion, neoforgeInstallerURL(loaderVersion), targetOS)
	default:
		return nil, fmt.Errorf("unsupported loader %q", loader)
	}
}

func (c *Client) buildVanillaManifest(ctx context.Context, instanceID, name, mcVersion, loader, targetOS string) (*InstanceLaunchManifest, error) {
	versionURL, err := c.ResolveVersionURL(ctx, mcVersion)
	if err != nil {
		return nil, err
	}
	meta, err := c.FetchVersionMeta(ctx, versionURL)
	if err != nil {
		return nil, err
	}
	return launchManifestFromMeta(instanceID, name, mcVersion, loader, "", versionURL, meta, targetOS), nil
}

func (c *Client) buildProfileManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion, profileURL, targetOS string) (*InstanceLaunchManifest, error) {
	meta, err := c.FetchVersionMeta(ctx, profileURL)
	if err != nil {
		return nil, err
	}
	meta, err = c.resolveInheritedMeta(ctx, meta)
	if err != nil {
		return nil, err
	}
	return launchManifestFromMeta(instanceID, name, mcVersion, loader, loaderVersion, profileURL, meta, targetOS), nil
}

func (c *Client) resolveInheritedMeta(ctx context.Context, meta *VersionMeta) (*VersionMeta, error) {
	if meta == nil {
		return nil, fmt.Errorf("missing version meta")
	}
	if meta.InheritsFrom == "" {
		return normalizeProfileMeta(meta), nil
	}
	parentURL, err := c.ResolveVersionURL(ctx, meta.InheritsFrom)
	if err != nil {
		return nil, err
	}
	parent, err := c.FetchVersionMeta(ctx, parentURL)
	if err != nil {
		return nil, err
	}
	parent, err = c.resolveInheritedMeta(ctx, parent)
	if err != nil {
		return nil, err
	}
	return normalizeProfileMeta(mergeVersionMeta(parent, meta)), nil
}

func mergeVersionMeta(parent, child *VersionMeta) *VersionMeta {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}
	merged := *parent
	if child.ID != "" {
		merged.ID = child.ID
	}
	if child.MainClass != "" {
		merged.MainClass = child.MainClass
	}
	if child.Arguments != nil {
		merged.Arguments = mergeVersionArguments(parent.Arguments, child.Arguments)
	}
	merged.Libraries = mergeLibraryLists(parent.Libraries, child.Libraries)
	if child.JavaVersion.MajorVersion > 0 {
		merged.JavaVersion = child.JavaVersion
	}
	if child.JavaVersion.Component != "" {
		merged.JavaVersion.Component = child.JavaVersion.Component
	}
	return &merged
}

func mergeVersionArguments(parent, child *VersionArguments) *VersionArguments {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}
	out := &VersionArguments{}
	if len(parent.Game) > 0 {
		out.Game = append(out.Game, parent.Game...)
	}
	if len(child.Game) > 0 {
		out.Game = append(out.Game, child.Game...)
	}
	if len(child.JVM) > 0 {
		out.JVM = child.JVM
	} else {
		out.JVM = parent.JVM
	}
	return out
}

func mergeLibraryLists(parent, child []Library) []Library {
	byName := make(map[string]Library, len(parent)+len(child))
	order := make([]string, 0, len(parent)+len(child))
	for _, lib := range parent {
		if _, ok := byName[lib.Name]; !ok {
			order = append(order, lib.Name)
		}
		byName[lib.Name] = lib
	}
	for _, lib := range child {
		if _, ok := byName[lib.Name]; !ok {
			order = append(order, lib.Name)
		}
		byName[lib.Name] = lib
	}
	out := make([]Library, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out
}

func normalizeProfileMeta(meta *VersionMeta) *VersionMeta {
	if meta == nil {
		return nil
	}
	out := *meta
	out.Libraries = normalizeLibraries(meta.Libraries)
	return &out
}

func normalizeLibraries(libs []Library) []Library {
	out := make([]Library, 0, len(libs))
	for _, lib := range libs {
		out = append(out, normalizeLibrary(lib))
	}
	return out
}

func normalizeLibrary(lib Library) Library {
	if lib.Downloads != nil && lib.Downloads.Artifact != nil && lib.Downloads.Artifact.URL != "" {
		return lib
	}
	if lib.RepoURL == "" {
		return lib
	}
	artifactURL := mavenArtifactURL(lib.RepoURL, lib.Name)
	if artifactURL == "" {
		return lib
	}
	lib.Downloads = &LibraryDownloads{
		Artifact: &DownloadFile{URL: artifactURL, Sha1: lib.Sha1},
	}
	return lib
}

func mavenArtifactURL(repo, name string) string {
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return ""
	}
	groupPath := strings.ReplaceAll(parts[0], ".", "/")
	artifact := parts[1]
	version := parts[2]
	suffix := ""
	if len(parts) >= 4 && parts[3] != "" {
		suffix = "-" + parts[3]
	}
	file := fmt.Sprintf("%s-%s%s.jar", artifact, version, suffix)
	return strings.TrimSuffix(repo, "/") + "/" + groupPath + "/" + artifact + "/" + version + "/" + file
}

func launchManifestFromMeta(instanceID, name, mcVersion, loader, loaderVersion, versionURL string, meta *VersionMeta, targetOS string) *InstanceLaunchManifest {
	gameArgs, jvmArgs := extractLaunchArguments(meta, targetOS)
	return &InstanceLaunchManifest{
		InstanceID:    instanceID,
		Name:          name,
		MCVersion:     mcVersion,
		Loader:        loader,
		LoaderVersion: loaderVersion,
		VersionID:     meta.ID,
		VersionURL:    versionURL,
		MainClass:     meta.MainClass,
		AssetIndex:    meta.AssetIndex,
		ClientJar:     meta.Downloads.Client,
		Libraries:     filterLibraries(normalizeLibraries(meta.Libraries), targetOS),
		JavaMajor:     meta.JavaVersion.MajorVersion,
		JavaComponent: meta.JavaVersion.Component,
		GameArguments: gameArgs,
		JVMArguments:  jvmArgs,
	}
}

func extractLaunchArguments(meta *VersionMeta, targetOS string) (game []string, jvm []string) {
	if meta == nil || meta.Arguments == nil {
		return nil, nil
	}
	return flattenArgumentList(meta.Arguments.Game, targetOS), flattenArgumentList(meta.Arguments.JVM, targetOS)
}

func flattenArgumentList(items []json.RawMessage, targetOS string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		out = append(out, parseArgumentEntry(raw, targetOS)...)
	}
	return out
}

func parseArgumentEntry(raw json.RawMessage, targetOS string) []string {
	if len(raw) == 0 {
		return nil
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return []string{asString}
	}
	var withRules struct {
		Rules []Rule            `json:"rules"`
		Value json.RawMessage   `json:"value"`
	}
	if err := json.Unmarshal(raw, &withRules); err != nil || len(withRules.Value) == 0 {
		return nil
	}
	if !rulesAllow(withRules.Rules, targetOS) {
		return nil
	}
	var valueString string
	if err := json.Unmarshal(withRules.Value, &valueString); err == nil {
		return []string{valueString}
	}
	var valueList []string
	if err := json.Unmarshal(withRules.Value, &valueList); err == nil {
		return valueList
	}
	return nil
}

