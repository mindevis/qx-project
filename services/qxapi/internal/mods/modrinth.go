package mods

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const modrinthAPIBase = "https://api.modrinth.com/v2"

type modrinthClient struct {
	httpClient *http.Client
	userAgent  string
}

type modrinthSearchResponse struct {
	Hits []struct {
		ProjectID      string   `json:"project_id"`
		Slug           string   `json:"slug"`
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		Author         string   `json:"author"`
		DisplayIconURL string   `json:"display_icon_url"`
		IconURL        string   `json:"icon_url"`
		ProjectType    string   `json:"project_type"`
		Downloads      int64    `json:"downloads"`
		Versions       []string `json:"versions"`
		Categories     []string `json:"categories"`
		ClientSide     string   `json:"client_side"`
		ServerSide     string   `json:"server_side"`
	} `json:"hits"`
}

type modrinthProject struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Body        string `json:"body"`
	IconURL     string `json:"icon_url"`
	Downloads   int64  `json:"downloads"`
	ProjectType string `json:"project_type"`
	ClientSide  string `json:"client_side"`
	ServerSide  string `json:"server_side"`
	Team        string `json:"team"`
}

type modrinthVersion struct {
	ID            string `json:"id"`
	VersionNumber string `json:"version_number"`
	GameVersions  []string
	Loaders       []string
	DatePublished string `json:"date_published"`
	Dependencies  []struct {
		VersionID      string `json:"version_id"`
		ProjectID      string `json:"project_id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
	Files []struct {
		Filename string `json:"filename"`
		URL      string `json:"url"`
		Size     int64  `json:"size"`
		Hashes   struct {
			SHA1 string `json:"sha1"`
		} `json:"hashes"`
	} `json:"files"`
}

func modrinthProjectType(projectType string) string {
	switch projectType {
	case ProjectTypeModpack, ProjectTypeResourcePack, ProjectTypeShader, ProjectTypeDatapack, ProjectTypePlugin:
		return projectType
	default:
		return ProjectTypeMod
	}
}

func modrinthExternalURL(slug string) string {
	return "https://modrinth.com/project/" + slug
}

func modrinthSearchIconURL(iconURL, displayIconURL string) string {
	if strings.TrimSpace(iconURL) != "" {
		return iconURL
	}
	return displayIconURL
}

func (c *modrinthClient) search(ctx context.Context, query, projectType, loader, mcVersion string, limit int) ([]SearchItem, error) {
	return c.searchProjects(ctx, query, projectType, loader, mcVersion, "relevance", limit, 0)
}

func (c *modrinthClient) browse(ctx context.Context, projectType, loader, mcVersion, sort string, limit, offset int) ([]SearchItem, error) {
	return c.searchProjects(ctx, "", projectType, loader, mcVersion, modrinthBrowseIndex(sort), limit, offset)
}

func (c *modrinthClient) searchProjects(
	ctx context.Context,
	query, projectType, loader, mcVersion, index string,
	limit, offset int,
) ([]SearchItem, error) {
	facets := [][]string{{"project_type:" + modrinthProjectTypeFacet(projectType)}}
	if loader != "" {
		facets = append(facets, []string{"categories:" + loaderFacetModrinth(loader)})
	}
	if mcVersion != "" {
		facets = append(facets, []string{"versions:" + mcVersion})
	}
	facetsJSON, err := json.Marshal(facets)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("facets", string(facetsJSON))
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))
	if index == "" {
		index = "downloads"
	}
	params.Set("index", index)

	var resp modrinthSearchResponse
	if err := c.getJSON(ctx, "/search?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]SearchItem, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		out = append(out, SearchItem{
			ID:           hit.ProjectID,
			Source:       SourceModrinth,
			Slug:         hit.Slug,
			Name:         hit.Title,
			Summary:      hit.Description,
			IconURL:      modrinthSearchIconURL(hit.IconURL, hit.DisplayIconURL),
			Downloads:    hit.Downloads,
			Author:       hit.Author,
			ProjectType:  modrinthProjectType(hit.ProjectType),
			Loaders:      hit.Categories,
			GameVersions: hit.Versions,
			ClientSide:   hit.ClientSide,
			ServerSide:   hit.ServerSide,
			ExternalURL:  modrinthExternalURL(hit.Slug),
		})
	}
	return out, nil
}

func modrinthBrowseIndex(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "newest":
		return "newest"
	case "updated":
		return "updated"
	case "relevance":
		return "relevance"
	default:
		return "downloads"
	}
}

func (c *modrinthClient) getProject(ctx context.Context, projectID string) (*ProjectDetail, error) {
	var p modrinthProject
	if err := c.getJSON(ctx, "/project/"+url.PathEscape(projectID), &p); err != nil {
		return nil, err
	}
	desc := strings.TrimSpace(p.Body)
	if desc == "" {
		desc = p.Description
	}
	return &ProjectDetail{
		SearchItem: SearchItem{
			ID:          p.ID,
			Source:      SourceModrinth,
			Slug:        p.Slug,
			Name:        p.Title,
			Summary:     p.Description,
			IconURL:     p.IconURL,
			Downloads:   p.Downloads,
			ProjectType: modrinthProjectType(p.ProjectType),
			ClientSide:  p.ClientSide,
			ServerSide:  p.ServerSide,
			ExternalURL: modrinthExternalURL(p.Slug),
		},
		Description: desc,
	}, nil
}

func (c *modrinthClient) listVersions(ctx context.Context, projectID, loader, mcVersion string) ([]Version, error) {
	path := "/project/" + url.PathEscape(projectID) + "/version"
	params := url.Values{}
	if loader != "" {
		params.Set("loaders", fmt.Sprintf(`["%s"]`, loaderFacetModrinth(loader)))
	}
	if mcVersion != "" {
		params.Set("game_versions", fmt.Sprintf(`["%s"]`, mcVersion))
	}
	if q := params.Encode(); q != "" {
		path += "?" + q
	}
	var raw []modrinthVersion
	if err := c.getJSON(ctx, path, &raw); err != nil {
		return nil, err
	}
	out := make([]Version, 0, len(raw))
	for _, v := range raw {
		out = append(out, *c.versionFromModrinthBasic(v))
	}
	return out, nil
}

func (c *modrinthClient) getVersion(ctx context.Context, versionID, loader, mcVersion string) (*Version, error) {
	var v modrinthVersion
	if err := c.getJSON(ctx, "/version/"+url.PathEscape(versionID), &v); err != nil {
		return nil, err
	}
	ver := c.versionFromModrinthBasic(v)
	deps, err := c.resolveDependencies(ctx, v.Dependencies, loader, mcVersion)
	if err != nil {
		return nil, err
	}
	ver.Dependencies = deps
	return ver, nil
}

func (c *modrinthClient) versionFromModrinthBasic(v modrinthVersion) *Version {
	files := make([]VersionFile, 0, len(v.Files))
	for _, f := range v.Files {
		files = append(files, VersionFile{
			Filename: f.Filename,
			URL:      f.URL,
			SHA1:     f.Hashes.SHA1,
			Size:     f.Size,
		})
	}
	return &Version{
		ID:            v.ID,
		VersionNumber: v.VersionNumber,
		GameVersions:  v.GameVersions,
		Loaders:       v.Loaders,
		Files:         files,
		PublishedAt:   v.DatePublished,
	}
}

func (c *modrinthClient) resolveDependencies(ctx context.Context, raw []struct {
	VersionID      string `json:"version_id"`
	ProjectID      string `json:"project_id"`
	DependencyType string `json:"dependency_type"`
}, loader, mcVersion string) ([]ModDependency, error) {
	out := make([]ModDependency, 0, len(raw))
	for _, dep := range raw {
		depType := normalizeDependencyType(dep.DependencyType)
		if depType == "embedded" {
			continue
		}
		entry := ModDependency{
			ProjectID:      dep.ProjectID,
			Source:         SourceModrinth,
			DependencyType: depType,
			VersionID:      dep.VersionID,
		}
		if dep.ProjectID != "" {
			if project, err := c.getProject(ctx, dep.ProjectID); err == nil {
				entry.ProjectName = project.Name
			}
		}
		if dep.VersionID != "" {
			var rv modrinthVersion
			if err := c.getJSON(ctx, "/version/"+url.PathEscape(dep.VersionID), &rv); err == nil {
				best := c.versionFromModrinthBasic(rv)
				entry.VersionNumber = best.VersionNumber
				if len(best.Files) > 0 {
					entry.Filename = best.Files[0].Filename
					entry.DownloadURL = best.Files[0].URL
					entry.FileSize = best.Files[0].Size
				}
			}
		} else if dep.ProjectID != "" {
			versions, err := c.listVersions(ctx, dep.ProjectID, loader, mcVersion)
			if err == nil && len(versions) > 0 {
				best := versions[0]
				entry.VersionID = best.ID
				entry.VersionNumber = best.VersionNumber
				if len(best.Files) > 0 {
					entry.Filename = best.Files[0].Filename
					entry.DownloadURL = best.Files[0].URL
					entry.FileSize = best.Files[0].Size
				}
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

func normalizeDependencyType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "optional":
		return "optional"
	case "embedded":
		return "embedded"
	default:
		return "required"
	}
}

func (c *modrinthClient) getJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modrinthAPIBase+path, nil)
	if err != nil {
		return err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("modrinth: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func modrinthProjectTypeFacet(projectType string) string {
	switch projectType {
	case ProjectTypeModpack:
		return "modpack"
	case ProjectTypeResourcePack:
		return "resourcepack"
	case ProjectTypeShader:
		return "shader"
	case ProjectTypeDatapack:
		return "datapack"
	case ProjectTypePlugin:
		return "plugin"
	default:
		return "mod"
	}
}

func loaderFacetModrinth(loader string) string {
	switch strings.ToLower(loader) {
	case "neoforge":
		return "neoforge"
	case "forge":
		return "forge"
	case "fabric":
		return "fabric"
	case "quilt":
		return "quilt"
	case "paper", "spigot", "purpur", "bukkit", "folia":
		return strings.ToLower(loader)
	case "mohist", "magma", "arclight":
		return "bukkit"
	default:
		return strings.ToLower(loader)
	}
}
