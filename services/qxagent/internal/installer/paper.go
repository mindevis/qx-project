package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/qxproject/qx/pkg/safepath"
)

const paperFillAPI = "https://fill.papermc.io/v3/projects/paper"

type paperBuildDownload struct {
	URL string `json:"url"`
}

type paperBuild struct {
	ID        int                           `json:"id"`
	Downloads map[string]paperBuildDownload `json:"downloads"`
}

func installPaper(ctx context.Context, opts Options, cfg InstallConfig) (StartSpec, error) {
	mcVersion := strings.TrimSpace(cfg.MCVersion)
	build := strings.TrimSpace(cfg.LoaderVersion)
	if mcVersion == "" || build == "" {
		return StartSpec{}, fmt.Errorf("paper install requires mc_version and loader_version")
	}
	if err := validateArtifact(mcVersion); err != nil {
		return StartSpec{}, fmt.Errorf("invalid mc version")
	}
	if err := validateArtifact(build); err != nil {
		return StartSpec{}, fmt.Errorf("invalid paper build")
	}

	workDir, err := safepath.ResolveRoot(cfg.WorkDir)
	if err != nil {
		return StartSpec{}, fmt.Errorf("work dir: %w", err)
	}
	javaBin, err := ensureJava(ctx, opts, mcVersion)
	if err != nil {
		return StartSpec{}, err
	}
	jarPath, err := safepath.Join(workDir, "server.jar")
	if err != nil {
		return StartSpec{}, err
	}
	if opts.DryRun {
		logLine(opts, "[QX] Paper install dry-run for "+mcVersion+" build "+build)
		return StartSpec{
			WorkDir: workDir,
			JarPath: jarPath,
			Args:    []string{"nogui"},
			JavaBin: javaBin,
		}, nil
	}

	logLine(opts, "[QX] Preparing Paper "+mcVersion+" #"+build+" in "+workDir)
	if err := safepath.EnsureDir(workDir); err != nil {
		return StartSpec{}, fmt.Errorf("mkdir work dir: %w", err)
	}
	downloadURL, err := resolvePaperDownloadURL(ctx, paperFillAPI, mcVersion, build)
	if err != nil {
		return StartSpec{}, err
	}
	logLine(opts, "[QX] Downloading Paper server jar…")
	downloadCtx, cancelDownload := context.WithTimeout(ctx, forgeDownloadTO)
	err = downloadFile(downloadCtx, downloadURL, jarPath)
	cancelDownload()
	if err != nil {
		return StartSpec{}, err
	}
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
	return StartSpec{
		WorkDir: workDir,
		JarPath: jarPath,
		Args:    []string{"nogui"},
		JavaBin: javaBin,
	}, nil
}

func resolvePaperDownloadURL(ctx context.Context, apiBase, mcVersion, build string) (string, error) {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	reqURL := apiBase + "/versions/" + mcVersion + "/builds"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", downloadUserAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("paper builds: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("paper builds: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("paper builds: http %d", res.StatusCode)
	}
	return paperJarURLFromBuilds(body, build)
}

func paperJarURLFromBuilds(body []byte, build string) (string, error) {
	want, err := strconv.Atoi(strings.TrimSpace(build))
	if err != nil {
		return "", fmt.Errorf("invalid paper build")
	}
	var builds []paperBuild
	if err := json.Unmarshal(body, &builds); err != nil {
		return "", fmt.Errorf("paper builds: %w", err)
	}
	for _, item := range builds {
		if item.ID != want {
			continue
		}
		if dl, ok := item.Downloads["server:default"]; ok && strings.TrimSpace(dl.URL) != "" {
			return dl.URL, nil
		}
		return "", fmt.Errorf("paper build %d has no server jar", want)
	}
	return "", fmt.Errorf("paper build %d not found", want)
}
