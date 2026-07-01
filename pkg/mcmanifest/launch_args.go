package mcmanifest

import (
	"strconv"
	"strings"
)

// ApplyExtraJVMArgs appends extra JVM flags after memory arguments.
func ApplyExtraJVMArgs(manifest *InstanceLaunchManifest, args []string) {
	if manifest == nil || len(args) == 0 {
		return
	}
	manifest.JVMArguments = append(manifest.JVMArguments, args...)
}

// ApplyWindowSize adds or replaces --width and --height in game arguments when set.
func ApplyWindowSize(manifest *InstanceLaunchManifest, width, height *int) {
	if manifest == nil {
		return
	}
	if width != nil && *width > 0 {
		setGameFlagValue(manifest, "--width", strconv.Itoa(*width))
	}
	if height != nil && *height > 0 {
		setGameFlagValue(manifest, "--height", strconv.Itoa(*height))
	}
}

func setGameFlagValue(manifest *InstanceLaunchManifest, flag, value string) {
	args := manifest.GameArguments
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			continue
		}
		if i+1 < len(args) {
			args[i+1] = value
		} else {
			args = append(args, value)
		}
		manifest.GameArguments = args
		return
	}
	manifest.GameArguments = append(args, flag, value)
}

// SanitizeJVMArgs trims whitespace and drops empty JVM argument lines.
func SanitizeJVMArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		out = append(out, arg)
	}
	return out
}
