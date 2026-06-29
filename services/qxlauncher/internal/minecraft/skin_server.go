package minecraft

import "strings"

// SkinServerConfig enables QX Skin Server session host override (Ely.by-style).
type SkinServerConfig struct {
	Enabled  bool
	HostBase string // e.g. https://api.example.com
}

// SkinServerJVMArgs returns JVM system properties that redirect the Minecraft session
// server to QX API for skin+cape profile lookups.
func SkinServerJVMArgs(hostBase string) []string {
	base := strings.TrimRight(strings.TrimSpace(hostBase), "/")
	if base == "" {
		return nil
	}
	return []string{
		"-Dminecraft.api.env=custom",
		"-Dminecraft.api.session.host=" + base + "/sessionserver",
	}
}

// PrependSkinServerJVMArgs inserts skin-server JVM flags before the main class / game args.
func PrependSkinServerJVMArgs(args []string, cfg SkinServerConfig) []string {
	if !cfg.Enabled {
		return args
	}
	jvm := SkinServerJVMArgs(cfg.HostBase)
	if len(jvm) == 0 {
		return args
	}
	insertAt := 0
	for insertAt < len(args) {
		arg := args[insertAt]
		if strings.HasPrefix(arg, "-X") || strings.HasPrefix(arg, "-D") {
			insertAt++
			continue
		}
		break
	}
	out := make([]string, 0, len(args)+len(jvm))
	out = append(out, args[:insertAt]...)
	out = append(out, jvm...)
	out = append(out, args[insertAt:]...)
	return out
}
