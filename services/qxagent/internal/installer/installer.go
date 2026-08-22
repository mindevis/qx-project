package installer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/mojangjava"
)

var ErrUnsupportedServerType = errors.New("unsupported server type for install")

const downloadUserAgent = "QXProject/1.0 (https://github.com/qxproject/qx)"

type StartSpec struct {
	WorkDir   string
	JarPath   string
	Command   string
	Args      []string
	JVMArgs   []string
	ExtraArgs []string
	JavaBin   string
}

type Options struct {
	DryRun           bool
	OnLog            func(line string)
	JavaRoot         string
	JavaPath         string
	SkipJavaDownload bool
}

func logLine(opts Options, line string) {
	if opts.OnLog != nil {
		opts.OnLog(line)
	}
}

func JavaRootFromServerRoot(serverRoot string) string {
	serverRoot = strings.TrimSpace(serverRoot)
	if serverRoot == "" {
		return "/opt/qxsystem/java"
	}
	return filepath.Join(filepath.Dir(serverRoot), "java")
}

func ensureJava(ctx context.Context, opts Options, mcVersion string) (string, error) {
	mgr := ensureJavaManager(opts)
	if !(opts.DryRun || opts.SkipJavaDownload) {
		logLine(opts, "[QX] Installing Mojang Java for Minecraft "+mcVersion+"…")
	}
	bin, err := mgr.EnsureForMcVersion(ctx, mcVersion)
	if err != nil {
		return "", fmt.Errorf("mojang java: %w", err)
	}
	if !(opts.DryRun || opts.SkipJavaDownload) {
		logLine(opts, "[QX] Java ready: "+bin)
	}
	return bin, nil
}

func ensureJavaMajor(ctx context.Context, opts Options, major int) (string, error) {
	mgr := ensureJavaManager(opts)
	if !(opts.DryRun || opts.SkipJavaDownload) {
		logLine(opts, fmt.Sprintf("[QX] Installing Mojang Java %d…", major))
	}
	bin, err := mgr.EnsureForRuntime(ctx, mojangjava.ComponentForMajor(major), major)
	if err != nil {
		return "", fmt.Errorf("mojang java: %w", err)
	}
	if !(opts.DryRun || opts.SkipJavaDownload) {
		logLine(opts, "[QX] Java ready: "+bin)
	}
	return bin, nil
}

func ensureJavaManager(opts Options) *mojangjava.Manager {
	javaRoot := strings.TrimSpace(opts.JavaRoot)
	if javaRoot == "" {
		javaRoot = "/opt/qxsystem/java"
	}
	return &mojangjava.Manager{
		RootDir:      javaRoot,
		JavaPath:     opts.JavaPath,
		SkipDownload: opts.DryRun || opts.SkipJavaDownload,
	}
}

type InstallConfig struct {
	ServerType    string
	WorkDir       string
	MCVersion     string
	LoaderVersion string
	Name          string
	Address       string
	Port          int
	RconPassword  string
}

func Install(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.ServerType)) {
	case "forge":
		return installForge(ctx, opts, cfg)
	case "neoforge":
		return installNeoForge(ctx, opts, cfg)
	case "paper":
		return installPaper(ctx, opts, cfg)
	case "velocity":
		return installVelocity(ctx, opts, cfg)
	default:
		return StartSpec{}, fmt.Errorf("%w: %s", ErrUnsupportedServerType, cfg.ServerType)
	}
}
