package minecraft

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ResolveJavaBin picks a Java executable: QX_JAVA → JAVA_HOME → PATH → "java".
func ResolveJavaBin() string {
	if custom := os.Getenv("QX_JAVA"); custom != "" {
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
