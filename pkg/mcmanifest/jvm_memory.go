package mcmanifest

import (
	"fmt"
	"strings"
)

// ApplyMaxMemoryMB overrides or prepends -Xmx in manifest JVM arguments.
func ApplyMaxMemoryMB(manifest *InstanceLaunchManifest, maxMB int) {
	if manifest == nil || maxMB <= 0 {
		return
	}
	xmx := formatXmx(maxMB)
	args := manifest.JVMArguments
	replaced := false
	for i, arg := range args {
		if strings.HasPrefix(arg, "-Xmx") {
			args[i] = xmx
			replaced = true
			break
		}
	}
	if !replaced {
		args = append([]string{xmx}, args...)
	}
	manifest.JVMArguments = args
}

func formatXmx(maxMB int) string {
	if maxMB >= 1024 && maxMB%1024 == 0 {
		return fmt.Sprintf("-Xmx%dG", maxMB/1024)
	}
	return fmt.Sprintf("-Xmx%dM", maxMB)
}
