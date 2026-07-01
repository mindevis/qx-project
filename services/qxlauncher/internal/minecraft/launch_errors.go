package minecraft

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LaunchFailureHint returns a short user-facing message for common launch failures.
func LaunchFailureHint(launchLogPath, gameDir string) string {
	text := readTail(launchLogPath, 64*1024)
	if text == "" {
		text = readTail(filepath.Join(gameDir, "logs", "latest.log"), 64*1024)
	}
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "unsatisfiedlinkerror") && strings.Contains(lower, "lwjgl"):
		return "не найдены LWJGL natives — проверьте папку natives и перезапустите"
	case strings.Contains(lower, "unsatisfiedlinkerror"):
		return "не найдены native-библиотеки — проверьте папку natives"
	case strings.Contains(lower, "could not find or load main class"):
		return "не найден главный класс — проверьте установку loader"
	default:
		if launchLogPath != "" {
			return "см. " + launchLogPath
		}
		return ""
	}
}

func readTail(path string, maxBytes int64) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	offset := int64(0)
	if info.Size() > maxBytes {
		offset = info.Size() - maxBytes
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return ""
	}
	b := make([]byte, info.Size()-offset)
	if _, err := io.ReadFull(f, b); err != nil {
		return ""
	}
	return string(b)
}
