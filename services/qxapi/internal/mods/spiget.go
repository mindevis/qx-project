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
	"time"
)

const spigetAPIBase = "https://api.spiget.org/v2"

type spigetNotFoundError struct{}

func (spigetNotFoundError) Error() string { return "spiget: not found" }

func isSpigetNotFound(err error) bool {
	_, ok := err.(spigetNotFoundError)
	return ok
}

type spigetClient struct {
	httpClient *http.Client
	userAgent  string
	apiBase    string
}

func (c *spigetClient) baseURL() string {
	if strings.TrimSpace(c.apiBase) != "" {
		return strings.TrimRight(c.apiBase, "/")
	}
	return spigetAPIBase
}

type spigetFile struct {
	Type        string  `json:"type"`
	Size        float64 `json:"size"`
	URL         string  `json:"url"`
	ExternalURL string  `json:"externalUrl"`
}

type spigetIcon struct {
	URL string `json:"url"`
}

type spigetAuthorRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type spigetResource struct {
	ID              int             `json:"id"`
	Name            string          `json:"name"`
	Tag             string          `json:"tag"`
	Downloads       int64           `json:"downloads"`
	Premium         bool            `json:"premium"`
	External        bool            `json:"external"`
	ExistenceStatus int             `json:"existenceStatus"`
	TestedVersions  []string        `json:"testedVersions"`
	File            spigetFile      `json:"file"`
	Icon            spigetIcon      `json:"icon"`
	Author          spigetAuthorRef `json:"author"`
	Version         struct {
		ID int `json:"id"`
	} `json:"version"`
}

type spigetVersion struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ReleaseDate int64  `json:"releaseDate"`
	Downloads   int64  `json:"downloads"`
	UUID        string `json:"uuid"`
}

func (c *spigetClient) search(ctx context.Context, query, projectType, _, _ string, limit int) ([]SearchItem, error) {
	if projectType != ProjectTypePlugin {
		return nil, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	params := url.Values{}
	params.Set("size", strconv.Itoa(clampSearchLimit(limit)))
	params.Set("page", "1")
	params.Set("sort", "-downloads")
	var resources []spigetResource
	if err := c.getJSON(ctx, "/search/resources/"+url.PathEscape(query)+"?"+params.Encode(), &resources); err != nil {
		if isSpigetNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return spigetSearchItems(resources), nil
}

func (c *spigetClient) browse(ctx context.Context, projectType, _, mcVersion, sort string, limit, offset int) ([]SearchItem, error) {
	if projectType != ProjectTypePlugin {
		return nil, nil
	}
	limit = clampSearchLimit(limit)
	page := offset/limit + 1
	if page < 1 {
		page = 1
	}
	params := url.Values{}
	params.Set("size", strconv.Itoa(limit))
	params.Set("page", strconv.Itoa(page))
	params.Set("sort", spigetSort(sort))
	path := "/resources/free?" + params.Encode()
	if version := strings.TrimSpace(mcVersion); version != "" {
		path = "/resources/for/" + url.PathEscape(version) + "?" + params.Encode()
	}
	var resources []spigetResource
	if err := c.getJSON(ctx, path, &resources); err != nil {
		if isSpigetNotFound(err) {
			resources = nil
		} else {
			return nil, err
		}
	}
	items := spigetSearchItems(resources)
	if len(items) > 0 || strings.TrimSpace(mcVersion) == "" {
		return items, nil
	}
	return c.browse(ctx, projectType, "", "", sort, limit, offset)
}

func (c *spigetClient) getProject(ctx context.Context, projectID string) (*ProjectDetail, error) {
	var resource spigetResource
	if err := c.getJSON(ctx, "/resources/"+url.PathEscape(projectID), &resource); err != nil {
		return nil, err
	}
	item := spigetSearchItem(resource)
	if item.ID == "" {
		return nil, fmt.Errorf("spiget: resource %s is not a free plugin", projectID)
	}
	if item.Author == "" && resource.Author.ID > 0 {
		item.Author = c.authorName(ctx, resource.Author.ID)
	}
	desc := strings.TrimSpace(resource.Tag)
	if body := c.resourceDescription(ctx, projectID); body != "" {
		desc = body
	}
	return &ProjectDetail{SearchItem: item, Description: desc}, nil
}

func (c *spigetClient) listVersions(ctx context.Context, projectID, _, mcVersion string) ([]Version, error) {
	resource, err := c.loadResource(ctx, projectID)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("size", "25")
	params.Set("page", "1")
	params.Set("sort", "-releaseDate")
	var raw []spigetVersion
	if err := c.getJSON(ctx, "/resources/"+url.PathEscape(projectID)+"/versions?"+params.Encode(), &raw); err != nil {
		return nil, err
	}
	out := make([]Version, 0, len(raw))
	for _, item := range raw {
		ver := spigetVersionFrom(resource, item)
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

func (c *spigetClient) getVersion(ctx context.Context, projectID, versionID, _, _ string) (*Version, error) {
	resource, err := c.loadResource(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var raw spigetVersion
	if err := c.getJSON(ctx, "/resources/"+url.PathEscape(projectID)+"/versions/"+url.PathEscape(versionID), &raw); err != nil {
		return nil, err
	}
	ver := spigetVersionFrom(resource, raw)
	if ver == nil {
		return nil, fmt.Errorf("spiget: version %s has no download", versionID)
	}
	return ver, nil
}

func (c *spigetClient) loadResource(ctx context.Context, projectID string) (spigetResource, error) {
	var resource spigetResource
	if err := c.getJSON(ctx, "/resources/"+url.PathEscape(projectID), &resource); err != nil {
		return spigetResource{}, err
	}
	if resource.Premium || resource.ExistenceStatus == 2 {
		return spigetResource{}, fmt.Errorf("spiget: resource %s is not downloadable", projectID)
	}
	return resource, nil
}

func (c *spigetClient) authorName(ctx context.Context, id int) string {
	var author spigetAuthorRef
	if err := c.getJSON(ctx, "/authors/"+strconv.Itoa(id), &author); err != nil {
		return ""
	}
	return strings.TrimSpace(author.Name)
}

func (c *spigetClient) resourceDescription(ctx context.Context, projectID string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+"/resources/"+url.PathEscape(projectID)+"/description", nil)
	if err != nil {
		return ""
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func spigetSearchItems(resources []spigetResource) []SearchItem {
	out := make([]SearchItem, 0, len(resources))
	for _, resource := range resources {
		if item := spigetSearchItem(resource); item.ID != "" {
			out = append(out, item)
		}
	}
	return out
}

func spigetSearchItem(resource spigetResource) SearchItem {
	if resource.ID <= 0 || resource.Premium || resource.ExistenceStatus == 2 {
		return SearchItem{}
	}
	if !spigetFileInstallable(resource) {
		return SearchItem{}
	}
	id := strconv.Itoa(resource.ID)
	return SearchItem{
		ID:           id,
		Source:       SourceSpigot,
		Slug:         spigetSlug(resource.Name, resource.ID),
		Name:         resource.Name,
		Summary:      resource.Tag,
		IconURL:      spigetIconURL(resource.Icon.URL),
		Downloads:    resource.Downloads,
		Author:       strings.TrimSpace(resource.Author.Name),
		ProjectType:  ProjectTypePlugin,
		GameVersions: resource.TestedVersions,
		ClientSide:   "unsupported",
		ServerSide:   "required",
		ExternalURL:  "https://www.spigotmc.org/resources/" + id + "/",
	}
}

func spigetVersionFrom(resource spigetResource, raw spigetVersion) *Version {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = strconv.Itoa(raw.ID)
	}
	fileURL := spigetDownloadURL(resource, raw.ID)
	if fileURL == "" {
		return nil
	}
	filename := filepath.Base(fileURL)
	if filename == "" || filename == "." || filename == "/" || !strings.Contains(filename, ".") {
		filename = resource.Name + "-" + name + ".jar"
	}
	published := ""
	if raw.ReleaseDate > 0 {
		published = time.Unix(raw.ReleaseDate, 0).UTC().Format(time.RFC3339)
	}
	return &Version{
		ID:            strconv.Itoa(raw.ID),
		VersionNumber: name,
		GameVersions:  resource.TestedVersions,
		Loaders:       []string{"paper", "spigot", "bukkit"},
		Files: []VersionFile{{
			Filename: filename,
			URL:      fileURL,
		}},
		PublishedAt: published,
	}
}

func spigetFileInstallable(resource spigetResource) bool {
	fileType := strings.ToLower(strings.TrimSpace(resource.File.Type))
	if fileType == ".sk" {
		return false
	}
	if resource.External {
		return looksLikePluginDownload(resource.File.ExternalURL)
	}
	return true
}

func spigetDownloadURL(resource spigetResource, versionID int) string {
	if looksLikePluginDownload(resource.File.ExternalURL) {
		return strings.TrimSpace(resource.File.ExternalURL)
	}
	if resource.External {
		return ""
	}
	base := spigetAPIBase
	// Version-specific /download redirects to spigotmc.org (Cloudflare 403).
	// Latest /download goes to cdn.spiget.org; older versions need /download/proxy.
	if versionID > 0 && versionID != resource.Version.ID {
		return fmt.Sprintf("%s/resources/%d/versions/%d/download/proxy", base, resource.ID, versionID)
	}
	return fmt.Sprintf("%s/resources/%d/download", base, resource.ID)
}

func looksLikePluginDownload(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	path := strings.ToLower(parsed.Path)
	if strings.HasSuffix(path, ".jar") || strings.HasSuffix(path, ".zip") {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "github.com" && strings.Contains(path, "/releases/download/")
}

func spigetIconURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "data:") {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return "https://www.spigotmc.org/" + strings.TrimPrefix(raw, "/")
}

func spigetSlug(name string, id int) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	if len(fields) == 0 {
		return strconv.Itoa(id)
	}
	slug := strings.Join(fields, "-")
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return -1
	}, slug)
	if slug == "" {
		return strconv.Itoa(id)
	}
	return slug
}

func spigetSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "newest":
		return "-releaseDate"
	case "updated":
		return "-updateDate"
	default:
		return "-downloads"
	}
}

func (c *spigetClient) getJSON(ctx context.Context, relPath string, dest any) error {
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
	if resp.StatusCode == http.StatusNotFound {
		return spigetNotFoundError{}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("spiget: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
