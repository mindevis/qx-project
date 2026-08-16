package minecraft

import (
	"os"
	"strings"

	"github.com/qxproject/qx/pkg/safepath"
)

const DefaultGameLanguage = "ru_ru"

// EnsureGameLanguage writes Minecraft options.txt so the client starts in the given language.
func EnsureGameLanguage(gameDir, lang string) error {
	if lang == "" {
		lang = DefaultGameLanguage
	}
	if err := safepath.EnsureDir(gameDir); err != nil {
		return err
	}

	path, err := safepath.Join(gameDir, "options.txt")
	if err != nil {
		return err
	}
	data, err := safepath.ReadFileBytes(path)
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
	return safepath.WriteFileBytes(path, []byte(out), 0o644)
}

func languageAssetKey(lang string) string {
	return "minecraft/lang/" + lang + ".json"
}
