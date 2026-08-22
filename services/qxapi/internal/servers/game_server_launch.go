package servers

import (
	"encoding/json"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const (
	minGameServerMemoryMB     = 512
	maxGameServerMemoryMB     = 65536
	defaultGameServerMemoryMB = 2048
	maxGameServerExtraArgs    = 64
)

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	n := *v
	return &n
}

func gameServerMemoryOrDefault(mb *int) int {
	if mb == nil || *mb <= 0 {
		return defaultGameServerMemoryMB
	}
	return *mb
}

func validateGameServerMemoryMB(mb int) error {
	if mb < minGameServerMemoryMB || mb > maxGameServerMemoryMB {
		return ErrValidation
	}
	return nil
}

func sanitizeGameServerExecArgs(args []string) ([]string, error) {
	clean := mcmanifest.SanitizeJVMArgs(args)
	if len(clean) > maxGameServerExtraArgs {
		return nil, ErrValidation
	}
	for _, arg := range clean {
		if !isSafeGameServerExecArg(arg) {
			return nil, ErrValidation
		}
	}
	return clean, nil
}

func isSafeGameServerExecArg(arg string) bool {
	if arg == "" || len(arg) > 4096 {
		return false
	}
	return !strings.ContainsAny(arg, "\x00\n\r;&|$`<>")
}

func stripMemoryJVMArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-Xms") || strings.HasPrefix(arg, "-Xmx") {
			continue
		}
		out = append(out, arg)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func installerStartArgs(item *models.GameServer) []string {
	if item == nil || strings.TrimSpace(item.StartArgsJSON) == "" {
		return nil
	}
	var args []string
	if err := json.Unmarshal([]byte(item.StartArgsJSON), &args); err != nil {
		return nil
	}
	return args
}

func gameServerJVMArgs(item *models.GameServer) []string {
	if item == nil {
		return nil
	}
	minMB := gameServerMemoryOrDefault(item.MinMemoryMB)
	maxMB := gameServerMemoryOrDefault(item.MaxMemoryMB)
	if minMB > maxMB {
		minMB = maxMB
	}
	manifest := &mcmanifest.InstanceLaunchManifest{}
	mcmanifest.ApplyMaxMemoryMB(manifest, maxMB)
	mcmanifest.ApplyMinMemoryMB(manifest, minMB)
	base := manifest.JVMArguments
	if !isProxyGameServerType(item.ServerType) {
		base = mcmanifest.MergeJVMArgs(base, mcmanifest.AikarJVMFlags())
	}
	extras := stripMemoryJVMArgs(mcmanifest.SanitizeJVMArgs([]string(item.ExtraJVMArgs)))
	return mcmanifest.MergeJVMArgs(base, extras)
}

func gameServerStartArgSets(item *models.GameServer) (args, jvmArgs, extraArgs []string) {
	if item == nil {
		return nil, nil, nil
	}
	jvmArgs = gameServerJVMArgs(item)
	installer := installerStartArgs(item)
	extra := append([]string{}, []string(item.ExtraArgs)...)
	if strings.TrimSpace(item.StartCommand) != "" {
		return installer, jvmArgs, extra
	}
	return nil, jvmArgs, append(append([]string{}, installer...), extra...)
}
