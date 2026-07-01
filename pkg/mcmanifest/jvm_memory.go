package mcmanifest

import (
	"fmt"
	"strings"
)

// ApplyMinMemoryMB overrides or prepends -Xms in manifest JVM arguments.
func ApplyMinMemoryMB(manifest *InstanceLaunchManifest, minMB int) {
	if manifest == nil || minMB <= 0 {
		return
	}
	xms := formatXms(minMB)
	args := manifest.JVMArguments
	replaced := false
	for i, arg := range args {
		if strings.HasPrefix(arg, "-Xms") {
			args[i] = xms
			replaced = true
			break
		}
	}
	if !replaced {
		args = append([]string{xms}, args...)
	}
	manifest.JVMArguments = args
}

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

func formatXms(minMB int) string {
	if minMB >= 1024 && minMB%1024 == 0 {
		return fmt.Sprintf("-Xms%dG", minMB/1024)
	}
	return fmt.Sprintf("-Xms%dM", minMB)
}

func formatXmx(maxMB int) string {
	if maxMB >= 1024 && maxMB%1024 == 0 {
		return fmt.Sprintf("-Xmx%dG", maxMB/1024)
	}
	return fmt.Sprintf("-Xmx%dM", maxMB)
}
