package mods

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Config holds upstream API credentials.
type Config struct {
	CurseForgeAPIKey  string
	ModrinthUserAgent string
}

// Service proxies CurseForge and Modrinth catalog APIs.
type Service struct {
	modrinth   *modrinthClient
	curseforge *curseForgeClient
}

func NewService(cfg Config) *Service {
	httpClient := http.DefaultClient
	ua := strings.TrimSpace(cfg.ModrinthUserAgent)
	if ua == "" {
		ua = "QXSystem/1.0 (https://github.com/qxproject/qx)"
	}
	return &Service{
		modrinth: &modrinthClient{httpClient: httpClient, userAgent: ua},
		curseforge: &curseForgeClient{
			httpClient: httpClient,
			apiKey:     strings.TrimSpace(cfg.CurseForgeAPIKey),
		},
	}
}

func (s *Service) CurseForgeEnabled() bool {
	return s.curseforge.enabled()
}

func (s *Service) Search(ctx context.Context, query, projectType, loader, mcVersion string, limit int) ([]SearchItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	projectType = normalizeProjectType(projectType)
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var (
		mrItems []SearchItem
		cfItems []SearchItem
		mrErr   error
		cfErr   error
		wg      sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		mrItems, mrErr = s.modrinth.search(ctx, query, projectType, loader, mcVersion, limit)
	}()
	go func() {
		defer wg.Done()
		cfItems, cfErr = s.curseforge.search(ctx, query, projectType, loader, mcVersion, limit)
	}()
	wg.Wait()

	if mrErr != nil && cfErr != nil {
		return nil, mrErr
	}
	if mrErr != nil {
		cfItems = cfItems[:min(len(cfItems), limit)]
		return cfItems, nil
	}
	if cfErr != nil || len(cfItems) == 0 {
		mrItems = mrItems[:min(len(mrItems), limit)]
		return mrItems, nil
	}

	merged := interleaveSearch(cfItems, mrItems, limit)
	return merged, nil
}

func (s *Service) Browse(ctx context.Context, projectType, loader, mcVersion, source, sort string, limit, offset int) ([]SearchItem, bool, error) {
	projectType = normalizeProjectType(projectType)
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	source = strings.ToLower(strings.TrimSpace(source))

	var (
		items   []SearchItem
		err     error
		hasMore bool
	)
	switch source {
	case SourceCurseForge:
		items, err = s.curseforge.browse(ctx, projectType, loader, mcVersion, sort, limit, offset)
	case SourceModrinth:
		items, err = s.modrinth.browse(ctx, projectType, loader, mcVersion, sort, limit, offset)
	default:
		items, err = s.browseBoth(ctx, projectType, loader, mcVersion, sort, limit, offset)
	}
	if err != nil {
		return nil, false, err
	}
	hasMore = len(items) >= limit
	return items, hasMore, nil
}

func (s *Service) browseBoth(ctx context.Context, projectType, loader, mcVersion, sort string, limit, offset int) ([]SearchItem, error) {
	pageSize := limit
	if pageSize < 1 {
		pageSize = 20
	}
	half := pageSize / 2
	if half < 1 {
		half = 1
	}
	cfOffset := offset / 2
	mrOffset := offset / 2

	var (
		cfItems []SearchItem
		mrItems []SearchItem
		cfErr   error
		mrErr   error
		wg      sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		cfItems, cfErr = s.curseforge.browse(ctx, projectType, loader, mcVersion, sort, half, cfOffset)
	}()
	go func() {
		defer wg.Done()
		mrItems, mrErr = s.modrinth.browse(ctx, projectType, loader, mcVersion, sort, pageSize-half, mrOffset)
	}()
	wg.Wait()

	if mrErr != nil && cfErr != nil {
		return nil, mrErr
	}
	if mrErr != nil {
		return cfItems[:min(len(cfItems), pageSize)], nil
	}
	if cfErr != nil || len(cfItems) == 0 {
		return mrItems[:min(len(mrItems), pageSize)], nil
	}
	return interleaveSearch(cfItems, mrItems, pageSize), nil
}

func (s *Service) GetProject(ctx context.Context, source, projectID string) (*ProjectDetail, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id required")
	}
	switch source {
	case SourceModrinth:
		return s.modrinth.getProject(ctx, projectID)
	case SourceCurseForge:
		return s.curseforge.getProject(ctx, projectID)
	default:
		return nil, fmt.Errorf("unknown source %q", source)
	}
}

func (s *Service) ListVersions(ctx context.Context, source, projectID, loader, mcVersion string) ([]Version, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id required")
	}
	switch source {
	case SourceModrinth:
		return s.modrinth.listVersions(ctx, projectID, loader, mcVersion)
	case SourceCurseForge:
		return s.curseforge.listVersions(ctx, projectID, loader, mcVersion)
	default:
		return nil, fmt.Errorf("unknown source %q", source)
	}
}

func normalizeProjectType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ProjectTypeModpack:
		return ProjectTypeModpack
	case ProjectTypeResourcePack, "resource-pack", "resource_pack":
		return ProjectTypeResourcePack
	case ProjectTypeShader, "shaders":
		return ProjectTypeShader
	default:
		return ProjectTypeMod
	}
}

func interleaveSearch(primary, secondary []SearchItem, limit int) []SearchItem {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	out := make([]SearchItem, 0, limit)
	i, j := 0, 0
	for len(out) < limit && (i < len(primary) || j < len(secondary)) {
		if i < len(primary) {
			out = append(out, primary[i])
			i++
			if len(out) >= limit {
				break
			}
		}
		if j < len(secondary) {
			out = append(out, secondary[j])
			j++
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
