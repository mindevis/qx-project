package mojangjava

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func ComponentForMajor(major int) string {
	switch {
	case major >= 21:
		return "java-runtime-delta"
	case major >= 17:
		return "java-runtime-gamma"
	case major >= 16:
		return "java-runtime-alpha"
	case major > 0:
		return "jre-legacy"
	default:
		return ""
	}
}

func PlatformKey() string {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "arm64":
			return "windows-arm64"
		case "386":
			return "windows-x86"
		default:
			return "windows-x64"
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-os-arm64"
		}
		return "mac-os"
	case "linux":
		if runtime.GOARCH == "386" {
			return "linux-i386"
		}
		return "linux"
	default:
		return runtime.GOOS
	}
}

func CachedJavaBin(dir string) (string, bool) {
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.exe"
	}
	bin := filepath.Join(dir, "bin", name)
	st, err := os.Stat(bin)
	if err != nil || st.IsDir() {
		return "", false
	}
	return bin, true
}

// ResolveSystemJava picks Java: override → JAVA_HOME → PATH → "java".
func ResolveSystemJava(javaPath string) string {
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
