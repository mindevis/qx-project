package mods

import (
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/ttlcache"
)

var (
	projectCache       = ttlcache.New[*ProjectDetail](10*time.Minute, 500)
	versionListCache   = ttlcache.New[[]Version](5*time.Minute, 500)
	versionDetailCache = ttlcache.New[*Version](10*time.Minute, 800)
)

func cacheKey(parts ...string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += "|"
		}
		out += part
	}
	return out
}
