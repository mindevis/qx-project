package mods

const (
	SourceCurseForge = "curseforge"
	SourceModrinth   = "modrinth"
)

const (
	ProjectTypeMod          = "mod"
	ProjectTypeModpack      = "modpack"
	ProjectTypeResourcePack = "resourcepack"
	ProjectTypeShader       = "shader"
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

// Version is a normalized project version.
type Version struct {
	ID            string        `json:"id"`
	VersionNumber string        `json:"version_number"`
	GameVersions  []string      `json:"game_versions,omitempty"`
	Loaders       []string      `json:"loaders,omitempty"`
	Files         []VersionFile `json:"files"`
	PublishedAt   string        `json:"published_at,omitempty"`
}

// SyncModRequest is the body for POST .../mods/sync.
type SyncModRequest struct {
	Source        string `json:"source"`
	ProjectID     string `json:"project_id"`
	VersionID     string `json:"version_id"`
	Filename      string `json:"filename"`
	DownloadURL   string `json:"download_url"`
	ProjectName   string `json:"project_name,omitempty"`
	VersionNumber string `json:"version_number,omitempty"`
}
