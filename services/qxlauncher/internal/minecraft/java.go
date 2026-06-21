package minecraft

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolveJavaBin picks Java: javaPath override → JAVA_HOME → PATH → "java".
func ResolveJavaBin(javaPath string) string {
	if custom := strings.TrimSpace(javaPath); custom != "" {
		return custom
	}
	if home := os.Getenv("JAVA_HOME"); home != "" {
		name := "java"
		if runtime.GOOS == "windows" {
			name = "java.exe"
		}
		candidate := filepath.Join(home, "bin", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if path, err := exec.LookPath("java"); err == nil {
		return path
	}
	return "java"
}
