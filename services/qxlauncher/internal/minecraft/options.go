package minecraft

import (
	"os"
	"path/filepath"
	"strings"
)

const DefaultGameLanguage = "ru_ru"

// EnsureGameLanguage writes Minecraft options.txt so the client starts in the given language.
func EnsureGameLanguage(gameDir, lang string) error {
	if lang == "" {
		lang = DefaultGameLanguage
	}
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(gameDir, "options.txt")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := []string{}
	if err == nil && len(data) > 0 {
		lines = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	}

	langLine := "lang:" + lang
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "lang:") {
			lines[i] = langLine
			found = true
			break
		}
	}
	if !found {
		lines = append([]string{langLine}, lines...)
	}

	out := strings.Join(lines, "\n")
	if out != "" {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

func languageAssetKey(lang string) string {
	return "minecraft/lang/" + lang + ".json"
}
