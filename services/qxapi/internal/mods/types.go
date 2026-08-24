package mods

import "strings"

const (
	SourceCurseForge = "curseforge"
	SourceModrinth   = "modrinth"
	SourceHangar     = "hangar"
	SourceSpigot     = "spigot"
	SourceBukkit     = "bukkit"
)

const (
	ProjectTypeMod          = "mod"
	ProjectTypeModpack      = "modpack"
	ProjectTypeResourcePack = "resourcepack"
	ProjectTypeShader       = "shader"
	ProjectTypeDatapack     = "datapack"
	ProjectTypePlugin       = "plugin"
)

// SearchItem is a normalized catalog entry from CurseForge or Modrinth.
type SearchItem struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	Summary      string   `json:"summary,omitempty"`
	IconURL      string   `json:"icon_url,omitempty"`
	Downloads    int64    `json:"downloads,omitempty"`
	Author       string   `json:"author,omitempty"`
	ProjectType  string   `json:"project_type"`
	Loaders      []string `json:"loaders,omitempty"`
	GameVersions []string `json:"game_versions,omitempty"`
	ClientSide   string   `json:"client_side,omitempty"`
	ServerSide   string   `json:"server_side,omitempty"`
	ExternalURL  string   `json:"external_url"`
}

// ProjectDetail extends SearchItem with description for detail views.
type ProjectDetail struct {
	SearchItem
	Description string `json:"description,omitempty"`
}

// VersionFile is a downloadable artifact for a project version.
type VersionFile struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	SHA1     string `json:"sha1,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// ModDependency is a resolved dependency for a mod version.
type ModDependency struct {
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name,omitempty"`
	Source         string `json:"source"`
	DependencyType string `json:"dependency_type"`
	VersionID      string `json:"version_id,omitempty"`
	VersionNumber  string `json:"version_number,omitempty"`
	Filename       string `json:"filename,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
	FileSize       int64  `json:"file_size,omitempty"`
}

// Version is a normalized project version.
type Version struct {
	ID            string          `json:"id"`
	VersionNumber string          `json:"version_number"`
	VersionType   string          `json:"version_type,omitempty"`
	GameVersions  []string        `json:"game_versions,omitempty"`
	Loaders       []string        `json:"loaders,omitempty"`
	Files         []VersionFile   `json:"files"`
	Dependencies  []ModDependency `json:"dependencies,omitempty"`
	PublishedAt   string          `json:"published_at,omitempty"`
}

// CatalogProjectUsesLoader reports whether mod-loader filters apply to a catalog project type.
func CatalogProjectUsesLoader(projectType string) bool {
	switch projectType {
	case ProjectTypeResourcePack, ProjectTypeShader, ProjectTypeDatapack:
		return false
	default:
		return true
	}
}

func IsVelocityLoader(loader string) bool {
	return strings.EqualFold(strings.TrimSpace(loader), "velocity")
}

func IsProxyPluginLoader(loader string) bool {
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "velocity", "waterfall", "bungeecord":
		return true
	default:
		return false
	}
}

func pluginCatalogAllowsBukkitSources(loader string) bool {
	return !IsVelocityLoader(loader)
}

func skipBukkitPluginSources(projectType, loader string) bool {
	return projectType == ProjectTypePlugin && !pluginCatalogAllowsBukkitSources(loader)
}

// SyncModRequest is the body for POST .../mods/sync.
type SyncModRequest struct {
	Source          string `json:"source"`
	ProjectID       string `json:"project_id"`
	VersionID       string `json:"version_id"`
	Filename        string `json:"filename"`
	DownloadURL     string `json:"download_url"`
	ProjectName     string `json:"project_name,omitempty"`
	VersionNumber   string `json:"version_number,omitempty"`
	VersionType     string `json:"version_type,omitempty"`
	ModTarget       string `json:"mod_target,omitempty"`
	SideOverride    string `json:"side_override,omitempty"`
	ReplaceFilename string `json:"replace_filename,omitempty"`
	IconURL         string `json:"icon_url,omitempty"`
	Downloads       int64  `json:"downloads,omitempty"`
	FileSize        int64  `json:"file_size,omitempty"`
}
