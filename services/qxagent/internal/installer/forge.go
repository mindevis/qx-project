package installer

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/safepath"
)

const (
	forgeMavenBase  = "https://maven.minecraftforge.net/net/minecraftforge/forge"
	forgeDownloadTO = 10 * time.Minute
	forgeInstallTO  = 20 * time.Minute
)

func installForge(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	workDir, err := safepath.ResolveRoot(cfg.WorkDir)
	if err != nil {
		return StartSpec{}, fmt.Errorf("work dir: %w", err)
	}
	mcVersion := strings.TrimSpace(cfg.MCVersion)
	loaderVersion := strings.TrimSpace(cfg.LoaderVersion)
	if mcVersion == "" || loaderVersion == "" {
		return StartSpec{}, fmt.Errorf("forge install requires mc_version and loader_version")
	}
	artifact := fmt.Sprintf("%s-%s", mcVersion, loaderVersion)
	if err := validateArtifact(artifact); err != nil {
		return StartSpec{}, err
	}
	javaBin, err := ensureJava(ctx, opts, mcVersion)
	if err != nil {
		return StartSpec{}, err
	}
	if opts.DryRun {
		logLine(opts, "[QX] Forge install dry-run for "+artifact)
		jarPath, err := safepath.Join(workDir, "forge-"+artifact+".jar")
		if err != nil {
			return StartSpec{}, err
		}
		return StartSpec{
			WorkDir: workDir,
			JarPath: jarPath,
			Args:    []string{"nogui"},
			JavaBin: javaBin,
		}, nil
	}

	logLine(opts, "[QX] Preparing Forge "+artifact+" in "+workDir)
	if err := safepath.EnsureDir(workDir); err != nil {
		return StartSpec{}, fmt.Errorf("mkdir work dir: %w", err)
	}

	installerURL := forgeInstallerURL(artifact)
	installerPath, err := safepath.Join(workDir, "forge-installer.jar")
	if err != nil {
		return StartSpec{}, err
	}
	logLine(opts, "[QX] Downloading installer…")
	downloadCtx, cancelDownload := context.WithTimeout(ctx, forgeDownloadTO)
	err = downloadFile(downloadCtx, installerURL, installerPath)
	cancelDownload()
	if err != nil {
		return StartSpec{}, err
	}
	logLine(opts, "[QX] Running Forge installer (may take several minutes)…")

	installCtx, cancelInstall := context.WithTimeout(ctx, forgeInstallTO)
	defer cancelInstall()
	cmd := exec.CommandContext(
		installCtx,
		javaBin,
		"-jar",
		installerPath,
		"--installServer",
	)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		tail := strings.TrimSpace(string(out))
		if len(tail) > 2000 {
			tail = tail[len(tail)-2000:]
		}
		return StartSpec{}, fmt.Errorf("forge installer: %w: %s", err, tail)
	}
	logLine(opts, "[QX] Forge installer finished")

	if err := acceptEULA(workDir); err != nil {
		return StartSpec{}, err
	}
	if err := configureServerProperties(workDir, ServerPropertiesConfig{
		Name:         cfg.Name,
		Address:      cfg.Address,
		Port:         cfg.Port,
		RconPassword: cfg.RconPassword,
	}); err != nil {
		return StartSpec{}, err
	}

	spec, err := forgeStartSpec(workDir, artifact, javaBin)
	if err != nil {
		return StartSpec{}, err
	}
	if spec.JavaBin == "" {
		spec.JavaBin = javaBin
	}
	return spec, nil
}

func validateArtifact(artifact string) error {
	if strings.TrimSpace(artifact) == "" {
		return fmt.Errorf("invalid artifact")
	}
	if strings.Contains(artifact, "..") || strings.Contains(artifact, "/") || strings.Contains(artifact, "\\") {
		return fmt.Errorf("invalid artifact")
	}
	return nil
}

func forgeInstallerURL(artifact string) string {
	return fmt.Sprintf("%s/%s/forge-%s-installer.jar", forgeMavenBase, artifact, artifact)
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 0}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: http %d", url, res.StatusCode)
	}
	return safepath.WriteStreamAtomic(dest, res.Body)
}

func acceptEULA(workDir string) error {
	path, err := safepath.Join(workDir, "eula.txt")
	if err != nil {
		return err
	}
	return safepath.WriteFileBytes(path, []byte("eula=true\n"), 0o644)
}

func forgeStartSpec(workDir, artifact, javaBin string) (StartSpec, error) {
	unixRel := filepath.ToSlash(filepath.Join("libraries", "net", "minecraftforge", "forge", artifact, "unix_args.txt"))
	unixPath, err := safepath.JoinRel(workDir, unixRel)
	if err != nil {
		return StartSpec{}, err
	}
	if _, err := safepath.Stat(unixPath); err == nil {
		if javaBin == "" {
			javaBin = "java"
		}
		return StartSpec{
			WorkDir: workDir,
			Command: javaBin,
			Args:    []string{"@user_jvm_args.txt", "@" + unixRel, "nogui"},
			JavaBin: javaBin,
		}, nil
	}
	runSh, err := safepath.Join(workDir, "run.sh")
	if err != nil {
		return StartSpec{}, err
	}
	if _, err := safepath.Stat(runSh); err == nil {
		if err := safepath.Chmod(runSh, 0o755); err != nil {
			return StartSpec{}, err
		}
		return StartSpec{
			WorkDir: workDir,
			Command: runSh,
			Args:    []string{"nogui"},
			JavaBin: javaBin,
		}, nil
	}
	jarPath, err := safepath.Join(workDir, "forge-"+artifact+".jar")
	if err != nil {
		return StartSpec{}, err
	}
	if _, err := safepath.Stat(jarPath); err != nil {
		return StartSpec{}, fmt.Errorf("forge server jar not found in %s", workDir)
	}
	return StartSpec{
		WorkDir: workDir,
		JarPath: jarPath,
		Args:    []string{"nogui"},
	}, nil
}
