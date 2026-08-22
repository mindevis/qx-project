package mcmanifest

import "strings"

// Aikar JVM flags for Minecraft servers: https://mcflags.emc.gs
// Memory (-Xms/-Xmx) is applied separately from the configured RAM.
var aikarJVMFlags = []string{
	"-XX:+AlwaysPreTouch",
	"-XX:+DisableExplicitGC",
	"-XX:+ParallelRefProcEnabled",
	"-XX:+PerfDisableSharedMem",
	"-XX:+UnlockExperimentalVMOptions",
	"-XX:+UseG1GC",
	"-XX:G1HeapRegionSize=8M",
	"-XX:G1HeapWastePercent=5",
	"-XX:G1MaxNewSizePercent=40",
	"-XX:G1MixedGCCountTarget=4",
	"-XX:G1MixedGCLiveThresholdPercent=90",
	"-XX:G1NewSizePercent=30",
	"-XX:G1RSetUpdatingPauseTimePercent=5",
	"-XX:G1ReservePercent=20",
	"-XX:InitiatingHeapOccupancyPercent=15",
	"-XX:MaxGCPauseMillis=200",
	"-XX:MaxTenuringThreshold=1",
	"-XX:SurvivorRatio=32",
	"-Dusing.aikars.flags=https://mcflags.emc.gs",
	"-Daikars.new.flags=true",
}

func AikarJVMFlags() []string {
	out := make([]string, len(aikarJVMFlags))
	copy(out, aikarJVMFlags)
	return out
}

func JVMArgKey(arg string) string {
	arg = strings.TrimSpace(arg)
	switch {
	case strings.HasPrefix(arg, "-Xms"):
		return "-Xms"
	case strings.HasPrefix(arg, "-Xmx"):
		return "-Xmx"
	case strings.HasPrefix(arg, "-XX:"):
		s := strings.TrimPrefix(arg, "-XX:")
		s = strings.TrimPrefix(s, "+")
		s = strings.TrimPrefix(s, "-")
		if i := strings.IndexByte(s, '='); i >= 0 {
			s = s[:i]
		}
		return "-XX:" + s
	case strings.HasPrefix(arg, "-D"):
		if i := strings.IndexByte(arg, '='); i >= 0 {
			return arg[:i]
		}
		return arg
	default:
		return arg
	}
}

// MergeJVMArgs appends extra flags after base, replacing an earlier flag with the same key.
func MergeJVMArgs(base, extra []string) []string {
	indexByKey := make(map[string]int, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	appendOrReplace := func(arg string) {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			return
		}
		key := JVMArgKey(arg)
		if i, ok := indexByKey[key]; ok {
			out[i] = arg
			return
		}
		indexByKey[key] = len(out)
		out = append(out, arg)
	}
	for _, arg := range base {
		appendOrReplace(arg)
	}
	for _, arg := range extra {
		appendOrReplace(arg)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
