package installer

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/qxproject/qx/pkg/mcproxy"
	"github.com/qxproject/qx/pkg/safepath"
)

const waterfallFillAPI = "https://fill.papermc.io/v3/projects/waterfall"

func installWaterfall(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	version := strings.TrimSpace(cfg.MCVersion)
	build := strings.TrimSpace(cfg.LoaderVersion)
	if version == "" || build == "" {
		return StartSpec{}, fmt.Errorf("waterfall install requires mc_version and loader_version")
	}
	if err := validateArtifact(version); err != nil {
		return StartSpec{}, fmt.Errorf("invalid waterfall version")
	}
	if err := validateArtifact(build); err != nil {
		return StartSpec{}, fmt.Errorf("invalid waterfall build")
	}

	workDir, err := safepath.ResolveRoot(cfg.WorkDir)
	if err != nil {
		return StartSpec{}, fmt.Errorf("work dir: %w", err)
	}
	javaBin, err := ensureJava(ctx, opts, version)
	if err != nil {
		return StartSpec{}, err
	}
	jarPath, err := safepath.Join(workDir, "server.jar")
	if err != nil {
		return StartSpec{}, err
	}
	if opts.DryRun {
		logLine(opts, "[QX] Waterfall install dry-run for "+version+" build "+build)
		return StartSpec{WorkDir: workDir, JarPath: jarPath, JavaBin: javaBin}, nil
	}

	logLine(opts, "[QX] Preparing Waterfall "+version+" #"+build+" in "+workDir)
	if err := safepath.EnsureDir(workDir); err != nil {
		return StartSpec{}, fmt.Errorf("mkdir work dir: %w", err)
	}
	downloadURL, err := resolvePaperDownloadURL(ctx, waterfallFillAPI, version, build)
	if err != nil {
		return StartSpec{}, err
	}
	logLine(opts, "[QX] Downloading Waterfall server jar…")
	downloadCtx, cancelDownload := context.WithTimeout(ctx, forgeDownloadTO)
	err = downloadFile(downloadCtx, downloadURL, jarPath)
	cancelDownload()
	if err != nil {
		return StartSpec{}, err
	}
	if err := writeBungeeConfig(workDir, cfg); err != nil {
		return StartSpec{}, err
	}
	return StartSpec{WorkDir: workDir, JarPath: jarPath, JavaBin: javaBin}, nil
}

func installBungeeCord(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	build := strings.TrimSpace(cfg.LoaderVersion)
	if build == "" {
		build = strings.TrimSpace(cfg.MCVersion)
	}
	if build == "" {
		return StartSpec{}, fmt.Errorf("bungeecord install requires a build")
	}
	if err := validateArtifact(build); err != nil {
		return StartSpec{}, fmt.Errorf("invalid bungeecord build")
	}

	workDir, err := safepath.ResolveRoot(cfg.WorkDir)
	if err != nil {
		return StartSpec{}, fmt.Errorf("work dir: %w", err)
	}
	javaBin, err := EnsureJavaMajor(ctx, opts, 21)
	if err != nil {
		return StartSpec{}, err
	}
	jarPath, err := safepath.Join(workDir, "server.jar")
	if err != nil {
		return StartSpec{}, err
	}
	if opts.DryRun {
		logLine(opts, "[QX] BungeeCord install dry-run for build "+build)
		return StartSpec{WorkDir: workDir, JarPath: jarPath, JavaBin: javaBin}, nil
	}

	logLine(opts, "[QX] Preparing BungeeCord #"+build+" in "+workDir)
	if err := safepath.EnsureDir(workDir); err != nil {
		return StartSpec{}, fmt.Errorf("mkdir work dir: %w", err)
	}
	logLine(opts, "[QX] Downloading BungeeCord server jar…")
	downloadCtx, cancelDownload := context.WithTimeout(ctx, forgeDownloadTO)
	err = downloadFile(downloadCtx, mcproxy.BungeeCordJarURL(build), jarPath)
	cancelDownload()
	if err != nil {
		return StartSpec{}, err
	}
	if err := writeBungeeConfig(workDir, cfg); err != nil {
		return StartSpec{}, err
	}
	return StartSpec{WorkDir: workDir, JarPath: jarPath, JavaBin: javaBin}, nil
}

func writeBungeeConfig(workDir string, cfg InstallConfig) error {
	bindHost := "0.0.0.0"
	if ip := net.ParseIP(strings.TrimSpace(cfg.Address)); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		bindHost = ip.String()
	}
	port := cfg.Port
	if port <= 0 {
		port = 25565
	}
	bind := bindHost + ":" + strconv.Itoa(port)
	path, err := safepath.Join(workDir, "config.yml")
	if err != nil {
		return err
	}
	return safepath.WriteFileBytes(path, []byte(mcproxy.BungeeConfigYAML(bind, strings.TrimSpace(cfg.Name), nil, nil)), 0o644)
}
