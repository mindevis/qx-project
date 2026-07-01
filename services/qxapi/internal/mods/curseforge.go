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

const (
	curseForgeAPIBase = "https://api.curseforge.com/v1"
	curseForgeGameID  = 432
)

type curseForgeClient struct {
	httpClient *http.Client
	apiKey     string
	apiBase    string // test override; empty uses curseForgeAPIBase
}

func (c *curseForgeClient) baseURL() string {
	if strings.TrimSpace(c.apiBase) != "" {
		return strings.TrimRight(c.apiBase, "/")
	}
	return curseForgeAPIBase
}

type curseForgeSearchResponse struct {
	Data []struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Summary       string `json:"summary"`
		DownloadCount int64  `json:"downloadCount"`
		Logo          struct {
			ThumbnailURL string `json:"thumbnailUrl"`
		} `json:"logo"`
		Authors []struct {
			Name string `json:"name"`
		} `json:"authors"`
		LatestFilesIndexes []struct {
			GameVersion string `json:"gameVersion"`
			ModLoader   int    `json:"modLoader"`
		} `json:"latestFilesIndexes"`
	} `json:"data"`
}

type curseForgeModResponse struct {
	Data struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		Summary       string `json:"summary"`
		Description   string `json:"description"`
		DownloadCount int64  `json:"downloadCount"`
		Logo          struct {
			ThumbnailURL string `json:"thumbnailUrl"`
		} `json:"logo"`
		Authors []struct {
			Name string `json:"name"`
		} `json:"authors"`
	} `json:"data"`
}

type curseForgeFilesResponse struct {
	Data []struct {
		ID            int    `json:"id"`
		DisplayName   string `json:"displayName"`
		FileName      string `json:"fileName"`
		FileDate      string `json:"fileDate"`
		GameVersions  []string
		ModLoader     int `json:"modLoader"`
		DownloadURL   string `json:"downloadUrl"`
		FileLength    int64  `json:"fileLength"`
		Hashes        []struct {
			Value string `json:"value"`
			Algo  int    `json:"algo"`
		} `json:"hashes"`
	} `json:"data"`
}

func (c *curseForgeClient) enabled() bool {
	return strings.TrimSpace(c.apiKey) != ""
}

func (c *curseForgeClient) search(ctx context.Context, query, projectType, loader, mcVersion string, limit int) ([]SearchItem, error) {
	return c.searchProjects(ctx, query, projectType, loader, mcVersion, "downloads", limit, 0)
}

func (c *curseForgeClient) browse(ctx context.Context, projectType, loader, mcVersion, sort string, limit, offset int) ([]SearchItem, error) {
	return c.searchProjects(ctx, "", projectType, loader, mcVersion, sort, limit, offset)
}

func (c *curseForgeClient) searchProjects(
	ctx context.Context,
	query, projectType, loader, mcVersion, sort string,
	limit, offset int,
) ([]SearchItem, error) {
	if !c.enabled() {
		return nil, nil
	}
	params := url.Values{}
	params.Set("gameId", strconv.Itoa(curseForgeGameID))
	params.Set("classId", strconv.Itoa(curseForgeClassID(projectType)))
	params.Set("searchFilter", query)
	params.Set("pageSize", strconv.Itoa(limit))
	params.Set("index", strconv.Itoa(offset))
	sortField, sortOrder := curseForgeSortParams(sort)
	params.Set("sortField", sortField)
	params.Set("sortOrder", sortOrder)
	if mcVersion != "" {
		params.Set("gameVersion", mcVersion)
	}
	if loaderType := curseForgeLoaderType(loader); loaderType != "" {
		params.Set("modLoaderType", loaderType)
	}
	var resp curseForgeSearchResponse
	if err := c.getJSON(ctx, "/mods/search?"+params.Encode(), &resp); err != nil {
		return nil, err
	}
	out := make([]SearchItem, 0, len(resp.Data))
	for _, item := range resp.Data {
		author := ""
		if len(item.Authors) > 0 {
			author = item.Authors[0].Name
		}
		versions := make([]string, 0, len(item.LatestFilesIndexes))
		loaders := make([]string, 0, len(item.LatestFilesIndexes))
		seenLoaders := map[string]struct{}{}
		for _, idx := range item.LatestFilesIndexes {
			if idx.GameVersion != "" {
				versions = append(versions, idx.GameVersion)
			}
			if l := curseForgeLoaderName(idx.ModLoader); l != "" {
				if _, ok := seenLoaders[l]; !ok {
					seenLoaders[l] = struct{}{}
					loaders = append(loaders, l)
				}
			}
		}
		out = append(out, SearchItem{
			ID:           strconv.Itoa(item.ID),
			Source:       SourceCurseForge,
			Slug:         item.Slug,
			Name:         item.Name,
			Summary:      item.Summary,
			IconURL:      item.Logo.ThumbnailURL,
			Downloads:    item.DownloadCount,
			Author:       author,
			ProjectType:  projectType,
			Loaders:      loaders,
			GameVersions: versions,
			ExternalURL:  curseForgeExternalURL(projectType, item.Slug),
		})
	}
	return out, nil
}

func (c *curseForgeClient) getProject(ctx context.Context, projectID string) (*ProjectDetail, error) {
	if !c.enabled() {
		return nil, fmt.Errorf("curseforge api key not configured")
	}
	var resp curseForgeModResponse
	if err := c.getJSON(ctx, "/mods/"+url.PathEscape(projectID), &resp); err != nil {
		return nil, err
	}
	item := resp.Data
	author := ""
	if len(item.Authors) > 0 {
		author = item.Authors[0].Name
	}
	return &ProjectDetail{
		SearchItem: SearchItem{
			ID:          strconv.Itoa(item.ID),
			Source:      SourceCurseForge,
			Slug:        item.Slug,
			Name:        item.Name,
			Summary:     item.Summary,
			IconURL:     item.Logo.ThumbnailURL,
			Downloads:   item.DownloadCount,
			Author:      author,
			ExternalURL: curseForgeExternalURL(ProjectTypeMod, item.Slug),
		},
		Description: item.Description,
	}, nil
}

func (c *curseForgeClient) listVersions(ctx context.Context, projectID, loader, mcVersion string) ([]Version, error) {
	if !c.enabled() {
		return nil, fmt.Errorf("curseforge api key not configured")
	}
	path := "/mods/" + url.PathEscape(projectID) + "/files"
	params := url.Values{}
	if mcVersion != "" {
		params.Set("gameVersion", mcVersion)
	}
	if loaderType := curseForgeLoaderType(loader); loaderType != "" {
		params.Set("modLoaderType", loaderType)
	}
	if q := params.Encode(); q != "" {
		path += "?" + q
	}
	var resp curseForgeFilesResponse
	if err := c.getJSON(ctx, path, &resp); err != nil {
		return nil, err
	}
	out := make([]Version, 0, len(resp.Data))
	for _, f := range resp.Data {
		sha1 := ""
		for _, h := range f.Hashes {
			if h.Algo == 1 {
				sha1 = h.Value
				break
			}
		}
		downloadURL := f.DownloadURL
		if downloadURL == "" {
			downloadURL, _ = c.fileDownloadURL(ctx, projectID, strconv.Itoa(f.ID))
		}
		out = append(out, Version{
			ID:            strconv.Itoa(f.ID),
			VersionNumber: f.DisplayName,
			GameVersions:  f.GameVersions,
			Loaders:       []string{curseForgeLoaderName(f.ModLoader)},
			Files: []VersionFile{{
				Filename: f.FileName,
				URL:      downloadURL,
				SHA1:     sha1,
				Size:     f.FileLength,
			}},
			PublishedAt: f.FileDate,
		})
	}
	return out, nil
}

func (c *curseForgeClient) fileDownloadURL(ctx context.Context, projectID, fileID string) (string, error) {
	var resp struct {
		Data string `json:"data"`
	}
	if err := c.getJSON(ctx, "/mods/"+url.PathEscape(projectID)+"/files/"+url.PathEscape(fileID)+"/download-url", &resp); err != nil {
		return "", err
	}
	return resp.Data, nil
}

func (c *curseForgeClient) getJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("curseforge: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func curseForgeSortParams(sort string) (field, order string) {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "newest", "updated":
		return "3", "desc"
	case "relevance":
		return "2", "desc"
	default:
		return "6", "desc"
	}
}

func curseForgeClassID(projectType string) int {
	switch projectType {
	case ProjectTypeModpack:
		return 4471
	case ProjectTypeResourcePack:
		return 12
	case ProjectTypeShader:
		return 6552
	default:
		return 6
	}
}

func curseForgeLoaderType(loader string) string {
	// CurseForge ModLoaderType enum: 1=Forge, 4=Fabric, 5=Quilt, 6=NeoForge (requires gameVersion).
	switch strings.ToLower(loader) {
	case "forge":
		return "1"
	case "neoforge":
		return "6"
	case "fabric":
		return "4"
	case "quilt":
		return "5"
	default:
		return ""
	}
}

func curseForgeLoaderName(code int) string {
	switch code {
	case 1:
		return "forge"
	case 4:
		return "fabric"
	case 5:
		return "quilt"
	case 6:
		return "neoforge"
	default:
		return ""
	}
}

func curseForgeExternalURL(projectType, slug string) string {
	segment := "mc-mods"
	switch projectType {
	case ProjectTypeModpack:
		segment = "modpacks"
	case ProjectTypeResourcePack:
		segment = "texture-packs"
	case ProjectTypeShader:
		segment = "shaders"
	}
	return "https://www.curseforge.com/minecraft/" + segment + "/" + slug
}
