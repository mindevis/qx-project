package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	forgeMavenBase     = "https://maven.minecraftforge.net/net/minecraftforge/forge"
	forgeDownloadTO    = 10 * time.Minute
	forgeInstallTO     = 20 * time.Minute
)

func installForge(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	workDir := cfg.WorkDir
	mcVersion := strings.TrimSpace(cfg.MCVersion)
	loaderVersion := strings.TrimSpace(cfg.LoaderVersion)
	if mcVersion == "" || loaderVersion == "" {
		return StartSpec{}, fmt.Errorf("forge install requires mc_version and loader_version")
	}
	artifact := fmt.Sprintf("%s-%s", mcVersion, loaderVersion)
	javaBin, err := ensureJava(ctx, opts, mcVersion)
	if err != nil {
		return StartSpec{}, err
	}
	if opts.DryRun {
		logLine(opts, "[QX] Forge install dry-run for "+artifact)
		return StartSpec{
			WorkDir: workDir,
			JarPath: filepath.Join(workDir, "forge-"+artifact+".jar"),
			Args:    []string{"nogui"},
			JavaBin: javaBin,
		}, nil
	}

	logLine(opts, "[QX] Preparing Forge "+artifact+" in "+workDir)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return StartSpec{}, fmt.Errorf("mkdir work dir: %w", err)
	}

	installerURL := forgeInstallerURL(artifact)
	installerPath := filepath.Join(workDir, "forge-installer.jar")
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
	tmp := dest + ".part"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, res.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func acceptEULA(workDir string) error {
	path := filepath.Join(workDir, "eula.txt")
	return os.WriteFile(path, []byte("eula=true\n"), 0o644)
}

func forgeStartSpec(workDir, artifact, javaBin string) (StartSpec, error) {
	unixRel := filepath.ToSlash(filepath.Join("libraries", "net", "minecraftforge", "forge", artifact, "unix_args.txt"))
	unixPath := filepath.Join(workDir, filepath.FromSlash(unixRel))
	if _, err := os.Stat(unixPath); err == nil {
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
	runSh := filepath.Join(workDir, "run.sh")
	if _, err := os.Stat(runSh); err == nil {
		if err := os.Chmod(runSh, 0o755); err != nil {
			return StartSpec{}, err
		}
		return StartSpec{
			WorkDir: workDir,
			Command: runSh,
			Args:    []string{"nogui"},
			JavaBin: javaBin,
		}, nil
	}
	jarPath := filepath.Join(workDir, "forge-"+artifact+".jar")
	if _, err := os.Stat(jarPath); err != nil {
		return StartSpec{}, fmt.Errorf("forge server jar not found in %s", workDir)
	}
	return StartSpec{
		WorkDir: workDir,
		JarPath: jarPath,
		Args:    []string{"nogui"},
	}, nil
}
