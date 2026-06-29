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
		ProjectType    string   `json:"project_type"`
		Downloads      int64    `json:"downloads"`
		Versions       []string `json:"versions"`
		Categories     []string `json:"categories"`
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
	Files         []struct {
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
	case ProjectTypeModpack, ProjectTypeResourcePack, ProjectTypeShader:
		return projectType
	default:
		return ProjectTypeMod
	}
}

func modrinthExternalURL(slug string) string {
	return "https://modrinth.com/project/" + slug
}

func (c *modrinthClient) search(ctx context.Context, query, projectType, loader, mcVersion string, limit int) ([]SearchItem, error) {
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
	params.Set("index", "relevance")

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
			IconURL:      hit.DisplayIconURL,
			Downloads:    hit.Downloads,
			Author:       hit.Author,
			ProjectType:  modrinthProjectType(hit.ProjectType),
			Loaders:      hit.Categories,
			GameVersions: hit.Versions,
			ExternalURL:  modrinthExternalURL(hit.Slug),
		})
	}
	return out, nil
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
		files := make([]VersionFile, 0, len(v.Files))
		for _, f := range v.Files {
			files = append(files, VersionFile{
				Filename: f.Filename,
				URL:      f.URL,
				SHA1:     f.Hashes.SHA1,
				Size:     f.Size,
			})
		}
		out = append(out, Version{
			ID:            v.ID,
			VersionNumber: v.VersionNumber,
			GameVersions:  v.GameVersions,
			Loaders:       v.Loaders,
			Files:         files,
			PublishedAt:   v.DatePublished,
		})
	}
	return out, nil
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
	default:
		return strings.ToLower(loader)
	}
}
