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

const velocityFillAPI = "https://fill.papermc.io/v3/projects/velocity"

// VelocityJavaMajor is the JDK needed to run a Velocity release.
// 3.x needs Java 21; 4.x is compiled for Java 25 (class file 69).
func VelocityJavaMajor(version string) int {
	version = strings.TrimSpace(version)
	n := 0
	for _, r := range version {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	if n >= 4 {
		return 25
	}
	return 21
}

func installVelocity(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	version := strings.TrimSpace(cfg.MCVersion)
	build := strings.TrimSpace(cfg.LoaderVersion)
	if version == "" || build == "" {
		return StartSpec{}, fmt.Errorf("velocity install requires mc_version and loader_version")
	}
	if err := validateArtifact(version); err != nil {
		return StartSpec{}, fmt.Errorf("invalid velocity version")
	}
	if err := validateArtifact(build); err != nil {
		return StartSpec{}, fmt.Errorf("invalid velocity build")
	}

	workDir, err := safepath.ResolveRoot(cfg.WorkDir)
	if err != nil {
		return StartSpec{}, fmt.Errorf("work dir: %w", err)
	}
	javaBin, err := EnsureJavaMajor(ctx, opts, VelocityJavaMajor(version))
	if err != nil {
		return StartSpec{}, err
	}
	jarPath, err := safepath.Join(workDir, "server.jar")
	if err != nil {
		return StartSpec{}, err
	}
	if opts.DryRun {
		logLine(opts, "[QX] Velocity install dry-run for "+version+" build "+build)
		return StartSpec{
			WorkDir: workDir,
			JarPath: jarPath,
			JavaBin: javaBin,
		}, nil
	}

	logLine(opts, "[QX] Preparing Velocity "+version+" #"+build+" in "+workDir)
	if err := safepath.EnsureDir(workDir); err != nil {
		return StartSpec{}, fmt.Errorf("mkdir work dir: %w", err)
	}
	downloadURL, err := resolvePaperDownloadURL(ctx, velocityFillAPI, version, build)
	if err != nil {
		return StartSpec{}, err
	}
	logLine(opts, "[QX] Downloading Velocity server jar…")
	downloadCtx, cancelDownload := context.WithTimeout(ctx, forgeDownloadTO)
	err = downloadFile(downloadCtx, downloadURL, jarPath)
	cancelDownload()
	if err != nil {
		return StartSpec{}, err
	}
	if err := writeVelocityConfig(workDir, cfg); err != nil {
		return StartSpec{}, err
	}
	return StartSpec{
		WorkDir: workDir,
		JarPath: jarPath,
		JavaBin: javaBin,
	}, nil
}

func writeVelocityConfig(workDir string, cfg InstallConfig) error {
	secret, err := mcproxy.GenerateForwardingSecret()
	if err != nil {
		return err
	}
	secretPath, err := safepath.Join(workDir, "forwarding.secret")
	if err != nil {
		return err
	}
	if err := safepath.WriteFileBytes(secretPath, []byte(secret+"\n"), 0o600); err != nil {
		return err
	}
	bindHost := "0.0.0.0"
	if ip := net.ParseIP(strings.TrimSpace(cfg.Address)); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		bindHost = ip.String()
	}
	port := cfg.Port
	if port <= 0 {
		port = 25565
	}
	bind := bindHost + ":" + strconv.Itoa(port)
	motd := strings.TrimSpace(cfg.Name)
	tomlPath, err := safepath.Join(workDir, "velocity.toml")
	if err != nil {
		return err
	}
	return safepath.WriteFileBytes(tomlPath, []byte(mcproxy.VelocityToml(bind, motd, nil, nil)), 0o644)
}
