package mods

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CurseForgeHTTPError is returned when the CurseForge API responds with a non-200 status.
type CurseForgeHTTPError struct {
	StatusCode int
	Body       string
}

func (e *CurseForgeHTTPError) Error() string {
	return fmt.Sprintf("curseforge: status %d: %s", e.StatusCode, strings.TrimSpace(e.Body))
}

func isCurseForgeHTTPStatus(err error, code int) bool {
	var cfErr *CurseForgeHTTPError
	return errors.As(err, &cfErr) && cfErr.StatusCode == code
}

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

type curseForgeLatestFile struct {
	GameVersions []string `json:"gameVersions"`
	ModLoader    int      `json:"modLoader"`
	IsServerPack bool     `json:"isServerPack"`
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
		LatestFiles []curseForgeLatestFile `json:"latestFiles"`
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
		LatestFiles []curseForgeLatestFile `json:"latestFiles"`
	} `json:"data"`
}

type curseForgeFileDependency struct {
	ModID        int `json:"modId"`
	RelationType int `json:"relationType"`
}

type curseForgeFileData struct {
	ID           int    `json:"id"`
	DisplayName  string `json:"displayName"`
	FileName     string `json:"fileName"`
	FileDate     string `json:"fileDate"`
	GameVersions []string
	ModLoader    int    `json:"modLoader"`
	DownloadURL  string `json:"downloadUrl"`
	FileLength   int64  `json:"fileLength"`
	Hashes       []struct {
		Value string `json:"value"`
		Algo  int    `json:"algo"`
	} `json:"hashes"`
	Dependencies []curseForgeFileDependency `json:"dependencies"`
}

type curseForgeFilesResponse struct {
	Data []curseForgeFileData `json:"data"`
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

func (c *curseForgeClient) browseStrict(ctx context.Context, projectType, loader, mcVersion, sort string, limit, offset int) ([]SearchItem, error) {
	if !c.enabled() {
		return nil, nil
	}
	return c.searchProjectsOnce(ctx, "", projectType, loader, mcVersion, sort, limit, offset)
}

func (c *curseForgeClient) searchProjects(
	ctx context.Context,
	query, projectType, loader, mcVersion, sort string,
	limit, offset int,
) ([]SearchItem, error) {
	if !c.enabled() {
		return nil, nil
	}
	started := time.Now()
	items, err := c.searchProjectsOnce(ctx, query, projectType, loader, mcVersion, sort, limit, offset)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 || query != "" {
		return items, nil
	}
	if time.Since(started) > catalogRelaxBudget {
		return items, nil
	}
	// Browse with strict filters often returns zero rows on CurseForge; relax filters progressively.
	if CatalogProjectUsesLoader(projectType) && loader != "" {
		if relaxed, err := c.searchProjectsOnce(ctx, query, projectType, "", mcVersion, sort, limit, offset); err != nil {
			return nil, err
		} else if len(relaxed) > 0 {
			return relaxed, nil
		}
	}
	if mcVersion != "" {
		relaxedLoader := loader
		if !CatalogProjectUsesLoader(projectType) {
			relaxedLoader = ""
		}
		return c.searchProjectsOnce(ctx, query, projectType, relaxedLoader, "", sort, limit, offset)
	}
	return items, nil
}

func (c *curseForgeClient) searchProjectsOnce(
	ctx context.Context,
	query, projectType, loader, mcVersion, sort string,
	limit, offset int,
) ([]SearchItem, error) {
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
	if CatalogProjectUsesLoader(projectType) {
		if loaderType := curseForgeLoaderType(loader); loaderType != "" {
			params.Set("modLoaderType", loaderType)
		}
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
			ClientSide:   curseForgeClientSide(item.LatestFiles, loader, mcVersion),
			ServerSide:   curseForgeServerSide(item.LatestFiles, loader, mcVersion),
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
			ClientSide:  curseForgeClientSide(item.LatestFiles, "", ""),
			ServerSide:  curseForgeServerSide(item.LatestFiles, "", ""),
			ExternalURL: curseForgeExternalURL(ProjectTypeMod, item.Slug),
		},
		Description: item.Description,
	}, nil
}

func (c *curseForgeClient) listVersions(ctx context.Context, projectID, loader, mcVersion string) ([]Version, error) {
	if !c.enabled() {
		return nil, fmt.Errorf("curseforge api key not configured")
	}
	items, err := c.listVersionsOnce(ctx, projectID, loader, mcVersion)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 || (loader == "" && mcVersion == "") {
		return items, nil
	}
	if loader != "" {
		items, err = c.listVersionsOnce(ctx, projectID, "", mcVersion)
		if err != nil || len(items) > 0 || mcVersion == "" {
			return items, err
		}
	}
	return c.listVersionsOnce(ctx, projectID, "", "")
}

func (c *curseForgeClient) listVersionsOnce(ctx context.Context, projectID, loader, mcVersion string) ([]Version, error) {
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
		if downloadURL == "" {
			continue
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

func (c *curseForgeClient) getVersion(ctx context.Context, projectID, fileID, loader, mcVersion string) (*Version, error) {
	if !c.enabled() {
		return nil, fmt.Errorf("curseforge api key not configured")
	}
	var resp struct {
		Data curseForgeFileData `json:"data"`
	}
	path := "/mods/" + url.PathEscape(projectID) + "/files/" + url.PathEscape(fileID)
	if err := c.getJSON(ctx, path, &resp); err != nil {
		if isCurseForgeHTTPStatus(err, http.StatusNotFound) {
			if version, findErr := c.findVersionInList(ctx, projectID, fileID, loader, mcVersion); findErr == nil {
				return version, nil
			}
		}
		return nil, err
	}
	return c.versionFromFileData(ctx, projectID, fileID, resp.Data, loader, mcVersion)
}

func (c *curseForgeClient) findVersionInList(ctx context.Context, projectID, fileID, loader, mcVersion string) (*Version, error) {
	versions, err := c.listVersions(ctx, projectID, loader, mcVersion)
	if err != nil {
		return nil, err
	}
	for _, version := range versions {
		if version.ID == fileID {
			return &version, nil
		}
	}
	return nil, &CurseForgeHTTPError{StatusCode: http.StatusNotFound, Body: "file not found"}
}

func (c *curseForgeClient) versionFromFileData(
	ctx context.Context,
	projectID, fileID string,
	f curseForgeFileData,
	loader, mcVersion string,
) (*Version, error) {
	sha1 := ""
	for _, h := range f.Hashes {
		if h.Algo == 1 {
			sha1 = h.Value
			break
		}
	}
	downloadURL := f.DownloadURL
	if downloadURL == "" {
		downloadURL, _ = c.fileDownloadURL(ctx, projectID, fileID)
	}
	if downloadURL == "" {
		return nil, fmt.Errorf("missing download URL for file %s", fileID)
	}
	deps, err := c.resolveFileDependencies(ctx, f.Dependencies, loader, mcVersion)
	if err != nil {
		return nil, err
	}
	return &Version{
		ID:            fileID,
		VersionNumber: f.DisplayName,
		GameVersions:  f.GameVersions,
		Loaders:       []string{curseForgeLoaderName(f.ModLoader)},
		Files: []VersionFile{{
			Filename: f.FileName,
			URL:      downloadURL,
			SHA1:     sha1,
			Size:     f.FileLength,
		}},
		Dependencies: deps,
		PublishedAt:  f.FileDate,
	}, nil
}

func (c *curseForgeClient) resolveFileDependencies(
	ctx context.Context,
	raw []curseForgeFileDependency,
	loader, mcVersion string,
) ([]ModDependency, error) {
	out := make([]ModDependency, 0, len(raw))
	for _, dep := range raw {
		depType := curseForgeRelationType(dep.RelationType)
		if depType == "embedded" {
			continue
		}
		modID := strconv.Itoa(dep.ModID)
		entry := ModDependency{
			ProjectID:      modID,
			Source:         SourceCurseForge,
			DependencyType: depType,
		}
		if project, err := c.getProject(ctx, modID); err == nil {
			entry.ProjectName = project.Name
		}
		versions, err := c.listVersions(ctx, modID, loader, mcVersion)
		if err == nil && len(versions) > 0 {
			best := versions[0]
			entry.VersionID = best.ID
			entry.VersionNumber = best.VersionNumber
			if len(best.Files) > 0 {
				entry.Filename = best.Files[0].Filename
				entry.DownloadURL = best.Files[0].URL
				entry.FileSize = best.Files[0].Size
			}
			if entry.DownloadURL == "" && entry.VersionID != "" {
				if ver, err := c.getVersion(ctx, modID, entry.VersionID, loader, mcVersion); err == nil && len(ver.Files) > 0 {
					entry.Filename = ver.Files[0].Filename
					entry.DownloadURL = ver.Files[0].URL
					entry.FileSize = ver.Files[0].Size
				}
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

func curseForgeRelationType(code int) string {
	switch code {
	case 2:
		return "optional"
	case 3:
		return "required"
	case 1:
		return "embedded"
	default:
		return "required"
	}
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
		return &CurseForgeHTTPError{StatusCode: resp.StatusCode, Body: string(body)}
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
	case ProjectTypeDatapack:
		return 6945
	case ProjectTypePlugin:
		return 5
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

func curseForgeFileCandidates(files []curseForgeLatestFile, loader, mcVersion string) []curseForgeLatestFile {
	if loader == "" && mcVersion == "" {
		return files
	}
	matched := make([]curseForgeLatestFile, 0, len(files))
	for _, file := range files {
		loaderName := curseForgeLoaderName(file.ModLoader)
		if loader != "" && loaderName != loader {
			continue
		}
		if mcVersion != "" && !slicesContains(file.GameVersions, mcVersion) {
			continue
		}
		matched = append(matched, file)
	}
	if len(matched) > 0 {
		return matched
	}
	return files
}

func curseForgeSidesFromGameVersions(versions []string) (clientSide, serverSide string) {
	hasClient := false
	hasServer := false
	for _, version := range versions {
		switch strings.ToLower(strings.TrimSpace(version)) {
		case "client":
			hasClient = true
		case "server":
			hasServer = true
		}
	}
	switch {
	case hasClient && hasServer:
		return "required", "required"
	case hasServer:
		return "unsupported", "required"
	case hasClient:
		return "required", "unsupported"
	default:
		return "", ""
	}
}

func curseForgeSidesFromFile(file curseForgeLatestFile) (clientSide, serverSide string) {
	client, server := curseForgeSidesFromGameVersions(file.GameVersions)
	if client != "" || server != "" {
		return client, server
	}
	if file.IsServerPack {
		return "unsupported", "required"
	}
	return "", ""
}

func curseForgeSidesFromFiles(files []curseForgeLatestFile, loader, mcVersion string) (clientSide, serverSide string) {
	if len(files) == 0 {
		return "", ""
	}
	for _, file := range curseForgeFileCandidates(files, loader, mcVersion) {
		if client, server := curseForgeSidesFromFile(file); client != "" || server != "" {
			return client, server
		}
	}
	return "", ""
}

func curseForgeClientSide(files []curseForgeLatestFile, loader, mcVersion string) string {
	client, _ := curseForgeSidesFromFiles(files, loader, mcVersion)
	return client
}

func curseForgeServerSide(files []curseForgeLatestFile, loader, mcVersion string) string {
	_, server := curseForgeSidesFromFiles(files, loader, mcVersion)
	return server
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
	case ProjectTypeDatapack:
		segment = "data-packs"
	case ProjectTypePlugin:
		segment = "bukkit-plugins"
	}
	return "https://www.curseforge.com/minecraft/" + segment + "/" + slug
}
