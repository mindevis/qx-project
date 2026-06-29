package cosmetics

import (
	"strings"

	"github.com/google/uuid"
)

// NormalizeGameUUID returns a canonical UUID string with dashes, or empty if invalid.
func NormalizeGameUUID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	compact := strings.ReplaceAll(raw, "-", "")
	if len(compact) != 32 {
		return ""
	}
	parsed, err := uuid.Parse(compact)
	if err != nil {
		return ""
	}
	return parsed.String()
}

// CompactGameUUID returns UUID without dashes (Mojang session id format).
func CompactGameUUID(raw string) string {
	normalized := NormalizeGameUUID(raw)
	if normalized == "" {
		return ""
	}
	return strings.ReplaceAll(normalized, "-", "")
}
