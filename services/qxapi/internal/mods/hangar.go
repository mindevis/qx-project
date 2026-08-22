package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

const hangarAPIBase = "https://hangar.papermc.io/api/v1"

type hangarClient struct {
	httpClient *http.Client
	userAgent  string
	apiBase    string
}

func (c *hangarClient) baseURL() string {
	if strings.TrimSpace(c.apiBase) != "" {
		return strings.TrimRight(c.apiBase, "/")
	}
	return hangarAPIBase
}

type hangarProjectsResponse struct {
	Result []hangarProject `json:"result"`
}

type hangarProject struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AvatarURL   string `json:"avatarUrl"`
	Namespace   struct {
		Owner string `json:"owner"`
		Slug  string `json:"slug"`
	} `json:"namespace"`
	Stats struct {
		Downloads int64 `json:"downloads"`
	} `json:"stats"`
	SupportedPlatforms map[string][]string `json:"supportedPlatforms"`
	MainPageContent    string              `json:"mainPageContent"`
}

type hangarVersionsResponse struct {
	Result []hangarVersion `json:"result"`
}

type hangarVersionDownload struct {
	FileInfo *struct {
		Name      string `json:"name"`
		SizeBytes int64  `json:"sizeBytes"`
		SHA256    string `json:"sha256Hash"`
	} `json:"fileInfo"`
	ExternalURL string `json:"externalUrl"`
	DownloadURL string `json:"downloadUrl"`
}

type hangarVersion struct {
	ID                   int                              `json:"id"`
	Name                 string                           `json:"name"`
	CreatedAt            string                           `json:"createdAt"`
	Downloads            map[string]hangarVersionDownload `json:"downloads"`
	PlatformDependencies map[string][]string              `json:"platformDependencies"`
}

func (c *hangarClient) search(ctx context.Context, query, projectType, loader, _ string, limit int) ([]SearchItem, error) {
	return c.searchProjects(ctx, query, projectType, loader, "", "relevance", limit, 0)
}

func (c *hangarClient) browse(ctx context.Context, projectType, loader, mcVersion, sort string, limit, offset int) ([]SearchItem, error) {
	items, err := c.searchProjects(ctx, "", projectType, loader, mcVersion, sort, limit, offset)
	if err != nil || len(items) > 0 || hangarGameVersion(loader, mcVersion) == "" {
		return items, err
	}
	return c.searchProjects(ctx, "", projectType, loader, "", sort, limit, offset)
}

func (c *hangarClient) searchProjects(
	ctx context.Context,
	query, projectType, loader, mcVersion, sort string,
	limit, offset int,
) ([]SearchItem, error) {
	if projectType != ProjectTypePlugin {
		return nil, nil
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(clampSearchLimit(limit)))
	params.Set("offset", strconv.Itoa(max(offset, 0)))
	params.Set("sort", hangarSort(sort))
	params.Set("platform", hangarPlatform(loader))
	if strings.TrimSpace(query) != "" {
		params.Set("q", strings.TrimSpace(query))
	} else if version := hangarGameVersion(loader, mcVersion); version != "" {
		params.Set("version", version)
	}
	var resp hangarProjectsResponse
	if err := c.getJSON(ctx, "/projects?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]SearchItem, 0, len(resp.Result))
	platform := hangarPlatform(loader)
	for _, project := range resp.Result {
		if !hangarSupportsPlatform(project, platform) {
			continue
		}
		if item := hangarSearchItem(project); item.ID != "" {
			out = append(out, item)
		}
	}
	return out, nil
}

func (c *hangarClient) getProject(ctx context.Context, projectID string) (*ProjectDetail, error) {
	slug := hangarProjectSlug(projectID)
	var project hangarProject
	if err := c.getJSON(ctx, "/projects/"+url.PathEscape(slug), &project); err != nil {
		return nil, err
	}
	item := hangarSearchItem(project)
	desc := strings.TrimSpace(project.MainPageContent)
	if desc == "" {
		desc = project.Description
	}
	return &ProjectDetail{SearchItem: item, Description: desc}, nil
}

func (c *hangarClient) listVersions(ctx context.Context, projectID, loader, mcVersion string) ([]Version, error) {
	slug := hangarProjectSlug(projectID)
	params := url.Values{}
	params.Set("limit", "25")
	params.Set("offset", "0")
	var resp hangarVersionsResponse
	if err := c.getJSON(ctx, "/projects/"+url.PathEscape(slug)+"/versions?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	platform := hangarPlatform(loader)
	out := make([]Version, 0, len(resp.Result))
	for _, raw := range resp.Result {
		ver := hangarVersionFrom(raw, platform)
		if ver == nil {
			continue
		}
		out = append(out, *ver)
	}
	if mcVersion != "" {
		matched := versionsMatchingGame(out, mcVersion)
		if len(matched) > 0 {
			return matched, nil
		}
	}
	return out, nil
}

func (c *hangarClient) getVersion(ctx context.Context, projectID, versionID, loader, mcVersion string) (*Version, error) {
	slug := hangarProjectSlug(projectID)
	var raw hangarVersion
	if err := c.getJSON(ctx, "/projects/"+url.PathEscape(slug)+"/versions/"+url.PathEscape(versionID), &raw); err != nil {
		return nil, err
	}
	ver := hangarVersionFrom(raw, hangarPlatform(loader))
	if ver == nil {
		return nil, fmt.Errorf("hangar: version %s has no download", versionID)
	}
	_ = mcVersion
	return ver, nil
}

func hangarSearchItem(project hangarProject) SearchItem {
	slug := strings.TrimSpace(project.Namespace.Slug)
	if slug == "" {
		return SearchItem{}
	}
	owner := strings.TrimSpace(project.Namespace.Owner)
	loaders := make([]string, 0, len(project.SupportedPlatforms))
	versions := make([]string, 0)
	for platform, gameVersions := range project.SupportedPlatforms {
		loaders = append(loaders, strings.ToLower(platform))
		versions = append(versions, gameVersions...)
	}
	return SearchItem{
		ID:           slug,
		Source:       SourceHangar,
		Slug:         slug,
		Name:         project.Name,
		Summary:      project.Description,
		IconURL:      project.AvatarURL,
		Downloads:    project.Stats.Downloads,
		Author:       owner,
		ProjectType:  ProjectTypePlugin,
		Loaders:      loaders,
		GameVersions: versions,
		ClientSide:   "unsupported",
		ServerSide:   "required",
		ExternalURL:  hangarExternalURL(owner, slug),
	}
}

func hangarVersionFrom(raw hangarVersion, platform string) *Version {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return nil
	}
	file, loaders, versions := hangarVersionFile(raw, platform)
	if file.URL == "" {
		return nil
	}
	return &Version{
		ID:            name,
		VersionNumber: name,
		GameVersions:  versions,
		Loaders:       loaders,
		Files:         []VersionFile{file},
		PublishedAt:   raw.CreatedAt,
	}
}

func hangarVersionFile(raw hangarVersion, preferred string) (VersionFile, []string, []string) {
	preferred = strings.ToUpper(strings.TrimSpace(preferred))
	order := []string{preferred}
	if preferred == "" || preferred == "PAPER" {
		order = []string{preferred, "PAPER", "WATERFALL", "VELOCITY"}
	}
	seen := map[string]struct{}{}
	try := make([]string, 0, len(order)+len(raw.Downloads))
	for _, platform := range order {
		platform = strings.ToUpper(strings.TrimSpace(platform))
		if platform == "" {
			continue
		}
		if _, ok := seen[platform]; ok {
			continue
		}
		seen[platform] = struct{}{}
		try = append(try, platform)
	}
	if preferred == "" || preferred == "PAPER" {
		for platform := range raw.Downloads {
			if _, ok := seen[platform]; ok {
				continue
			}
			try = append(try, platform)
		}
	}
	loaders := make([]string, 0, len(raw.Downloads))
	for platform := range raw.Downloads {
		loaders = append(loaders, strings.ToLower(platform))
	}
	for _, platform := range try {
		dl, ok := raw.Downloads[platform]
		if !ok {
			continue
		}
		fileURL := strings.TrimSpace(dl.DownloadURL)
		if fileURL == "" {
			fileURL = strings.TrimSpace(dl.ExternalURL)
		}
		if fileURL == "" {
			continue
		}
		filename := filepath.Base(fileURL)
		var size int64
		if dl.FileInfo != nil {
			if name := strings.TrimSpace(dl.FileInfo.Name); name != "" {
				filename = name
			}
			size = dl.FileInfo.SizeBytes
		}
		return VersionFile{Filename: filename, URL: fileURL, Size: size}, loaders, raw.PlatformDependencies[platform]
	}
	return VersionFile{}, loaders, nil
}

func hangarExternalURL(owner, slug string) string {
	if owner == "" {
		return "https://hangar.papermc.io/" + slug
	}
	return "https://hangar.papermc.io/" + url.PathEscape(owner) + "/" + url.PathEscape(slug)
}

func hangarProjectSlug(projectID string) string {
	projectID = strings.TrimSpace(projectID)
	if i := strings.LastIndex(projectID, "/"); i >= 0 {
		return projectID[i+1:]
	}
	return projectID
}

func hangarPlatform(loader string) string {
	switch strings.ToLower(strings.TrimSpace(loader)) {
	case "velocity":
		return "VELOCITY"
	case "waterfall", "bungeecord":
		return "WATERFALL"
	default:
		return "PAPER"
	}
}

func hangarGameVersion(loader, mcVersion string) string {
	if IsProxyPluginLoader(loader) {
		return ""
	}
	return strings.TrimSpace(mcVersion)
}

func hangarSupportsPlatform(project hangarProject, platform string) bool {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	if platform == "" || len(project.SupportedPlatforms) == 0 {
		return true
	}
	for name := range project.SupportedPlatforms {
		if strings.EqualFold(name, platform) {
			return true
		}
	}
	return false
}

func hangarSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "newest", "updated":
		return "-updated"
	case "relevance":
		return "-stars"
	default:
		return "-downloads"
	}
}

func versionsMatchingGame(versions []Version, mcVersion string) []Version {
	wanted := map[string]struct{}{strings.TrimSpace(mcVersion): {}}
	matched := make([]Version, 0, len(versions))
	for _, ver := range versions {
		for _, game := range ver.GameVersions {
			if _, ok := wanted[game]; ok {
				matched = append(matched, ver)
				break
			}
		}
	}
	return matched
}

func (c *hangarClient) getJSON(ctx context.Context, relPath string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+relPath, nil)
	if err != nil {
		return err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("hangar: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
