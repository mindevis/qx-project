package protocol

import "strings"

// ContentDisabledSuffix is the Minecraft convention for skipping a jar/zip on load.
const ContentDisabledSuffix = ".disabled"

// IsContentDisabledFilename reports whether name uses the disabled suffix and has a stem.
func IsContentDisabledFilename(name string) bool {
	name = strings.TrimSpace(name)
	n := len(ContentDisabledSuffix)
	return len(name) > n && strings.EqualFold(name[len(name)-n:], ContentDisabledSuffix)
}

// EnabledContentFilename strips a trailing .disabled suffix, if present.
func EnabledContentFilename(name string) string {
	name = strings.TrimSpace(name)
	if IsContentDisabledFilename(name) {
		return name[:len(name)-len(ContentDisabledSuffix)]
	}
	return name
}

// DisabledContentFilename returns the on-disk name for a disabled content file.
func DisabledContentFilename(name string) string {
	name = EnabledContentFilename(name)
	if name == "" {
		return ""
	}
	return name + ContentDisabledSuffix
}

// SameContentFilename reports whether two filenames refer to the same content file,
// ignoring a trailing .disabled suffix and letter case.
func SameContentFilename(a, b string) bool {
	left := EnabledContentFilename(a)
	return left != "" && strings.EqualFold(left, EnabledContentFilename(b))
}
