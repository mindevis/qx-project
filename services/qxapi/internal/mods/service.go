package mods

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/ttlcache"
)

// Config holds upstream API credentials.
type Config struct {
	CurseForgeAPIKey  string
	ModrinthUserAgent string
}

// Service proxies CurseForge and Modrinth catalog APIs.
type Service struct {
	modrinth    *modrinthClient
	curseforge  *curseForgeClient
	browseCache *ttlcache.Cache[[]SearchItem]
	searchCache *ttlcache.Cache[[]SearchItem]
}

func NewService(cfg Config) *Service {
	httpClient := newCatalogHTTPClient()
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
		browseCache: ttlcache.New[[]SearchItem](2*time.Minute, 200),
		searchCache: ttlcache.New[[]SearchItem](2*time.Minute, 200),
	}
}

func (s *Service) CurseForgeEnabled() bool {
	return s.curseforge.enabled()
}

func (s *Service) Search(ctx context.Context, query, projectType, loader, mcVersion, source string, limit int) ([]SearchItem, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	projectType = normalizeProjectType(projectType)
	if limit <= 0 || limit > maxSearchItems {
		limit = 20
	}
	source = strings.ToLower(strings.TrimSpace(source))
	load := func() ([]SearchItem, error) {
		return s.searchUncached(ctx, query, projectType, loader, mcVersion, source, limit)
	}
	if s.searchCache == nil {
		return load()
	}
	return s.searchCache.GetOrLoad(
		cacheKey("search", query, projectType, loader, mcVersion, source, strconv.Itoa(limit)),
		load,
	)
}

func (s *Service) searchUncached(ctx context.Context, query, projectType, loader, mcVersion, source string, limit int) ([]SearchItem, error) {
	switch source {
	case SourceCurseForge:
		if !s.curseforge.enabled() {
			return nil, fmt.Errorf("curseforge api key not configured")
		}
		return s.curseforge.search(ctx, query, projectType, loader, mcVersion, limit)
	case SourceModrinth:
		return s.modrinth.search(ctx, query, projectType, loader, mcVersion, limit)
	default:
		return s.searchUpstream(ctx, query, projectType, loader, mcVersion, limit)
	}
}

func (s *Service) searchUpstream(ctx context.Context, query, projectType, loader, mcVersion string, limit int) ([]SearchItem, error) {
	mrCh := runCatalogHalf(func() ([]SearchItem, error) {
		return s.modrinth.search(ctx, query, projectType, loader, mcVersion, limit)
	})
	cfCh := runCatalogHalf(func() ([]SearchItem, error) {
		return s.curseforge.search(ctx, query, projectType, loader, mcVersion, limit)
	})
	mr, cf, cfOK := waitPrimaryThenPartner(ctx, mrCh, cfCh, catalogPartnerGrace)
	return mergeCatalogHalves(mr, cf, cfOK, limit)
}

func (s *Service) Browse(ctx context.Context, projectType, loader, mcVersion, source, sort string, limit, offset int) ([]SearchItem, bool, error) {
	projectType = normalizeProjectType(projectType)
	if limit <= 0 || limit > maxSearchItems {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	source = strings.ToLower(strings.TrimSpace(source))
	load := func() ([]SearchItem, error) {
		return s.browseUncached(ctx, projectType, loader, mcVersion, source, sort, limit, offset)
	}
	var (
		items []SearchItem
		err   error
	)
	if s.browseCache == nil {
		items, err = load()
	} else {
		items, err = s.browseCache.GetOrLoad(
			cacheKey("browse", projectType, loader, mcVersion, source, sort, strconv.Itoa(limit), strconv.Itoa(offset)),
			load,
		)
	}
	if err != nil {
		return nil, false, err
	}
	return items, len(items) >= limit, nil
}

func (s *Service) browseUncached(ctx context.Context, projectType, loader, mcVersion, source, sort string, limit, offset int) ([]SearchItem, error) {
	switch source {
	case SourceCurseForge:
		if !s.curseforge.enabled() {
			return nil, fmt.Errorf("curseforge api key not configured")
		}
		return s.curseforge.browse(ctx, projectType, loader, mcVersion, sort, limit, offset)
	case SourceModrinth:
		return s.modrinth.browse(ctx, projectType, loader, mcVersion, sort, limit, offset)
	default:
		return s.browseBoth(ctx, projectType, loader, mcVersion, sort, limit, offset)
	}
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

	mrCh := runCatalogHalf(func() ([]SearchItem, error) {
		return s.modrinth.browse(ctx, projectType, loader, mcVersion, sort, pageSize, offset)
	})
	cfCh := runCatalogHalf(func() ([]SearchItem, error) {
		return s.curseforge.browseStrict(ctx, projectType, loader, mcVersion, sort, half, offset/2)
	})
	mr, cf, cfOK := waitPrimaryThenPartner(ctx, mrCh, cfCh, catalogPartnerGrace)
	return mergeCatalogHalves(mr, cf, cfOK, pageSize)
}

func mergeCatalogHalves(primary, partner catalogHalf, partnerOK bool, limit int) ([]SearchItem, error) {
	if !partnerOK {
		if primary.err != nil {
			return nil, primary.err
		}
		return primary.items[:min(len(primary.items), limit)], nil
	}
	if primary.err != nil && partner.err != nil {
		return nil, primary.err
	}
	if primary.err != nil {
		return partner.items[:min(len(partner.items), limit)], nil
	}
	if partner.err != nil || len(partner.items) == 0 {
		return primary.items[:min(len(primary.items), limit)], nil
	}
	return interleaveSearch(partner.items, primary.items, limit), nil
}

func (s *Service) GetProject(ctx context.Context, source, projectID string) (*ProjectDetail, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id required")
	}
	key := cacheKey("project", source, projectID)
	return projectCache.GetOrLoad(key, func() (*ProjectDetail, error) {
		switch source {
		case SourceModrinth:
			return s.modrinth.getProject(ctx, projectID)
		case SourceCurseForge:
			return s.curseforge.getProject(ctx, projectID)
		default:
			return nil, fmt.Errorf("unknown source %q", source)
		}
	})
}

func (s *Service) ListVersions(ctx context.Context, source, projectID, loader, mcVersion string) ([]Version, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("project id required")
	}
	key := cacheKey("versions", source, projectID, loader, mcVersion)
	return versionListCache.GetOrLoad(key, func() ([]Version, error) {
		switch source {
		case SourceModrinth:
			return s.modrinth.listVersions(ctx, projectID, loader, mcVersion)
		case SourceCurseForge:
			return s.curseforge.listVersions(ctx, projectID, loader, mcVersion)
		default:
			return nil, fmt.Errorf("unknown source %q", source)
		}
	})
}

func (s *Service) GetVersion(ctx context.Context, source, projectID, versionID, loader, mcVersion string) (*Version, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	projectID = strings.TrimSpace(projectID)
	versionID = strings.TrimSpace(versionID)
	if projectID == "" || versionID == "" {
		return nil, fmt.Errorf("project id and version id required")
	}
	key := cacheKey("version", source, projectID, versionID, loader, mcVersion)
	return versionDetailCache.GetOrLoad(key, func() (*Version, error) {
		switch source {
		case SourceModrinth:
			return s.modrinth.getVersion(ctx, versionID, loader, mcVersion)
		case SourceCurseForge:
			return s.curseforge.getVersion(ctx, projectID, versionID, loader, mcVersion)
		default:
			return nil, fmt.Errorf("unknown source %q", source)
		}
	})
}

func normalizeProjectType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ProjectTypeModpack:
		return ProjectTypeModpack
	case ProjectTypeResourcePack, "resource-pack", "resource_pack":
		return ProjectTypeResourcePack
	case ProjectTypeShader, "shaders":
		return ProjectTypeShader
	case ProjectTypeDatapack, "data-pack", "data_pack":
		return ProjectTypeDatapack
	case ProjectTypePlugin, "plugins":
		return ProjectTypePlugin
	default:
		return ProjectTypeMod
	}
}

const maxSearchItems = 50

func clampSearchLimit(limit int) int {
	if limit <= 0 || limit > maxSearchItems {
		return 20
	}
	return limit
}

func interleaveSearch(primary, secondary []SearchItem, limit int) []SearchItem {
	limit = clampSearchLimit(limit)
	out := make([]SearchItem, 0, maxSearchItems)
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
