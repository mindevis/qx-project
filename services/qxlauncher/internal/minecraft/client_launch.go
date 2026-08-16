package minecraft

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/pkg/safepath"
	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

type LaunchAuth struct {
	Username    string
	UUID        string
	AccessToken string
}

type LaunchCosmetics struct {
	SkinModel      string
	SkinURL        string
	UseSkinServer  bool
	SkinServerHost string
	GameUUID       string
}

type ClientLaunchInput struct {
	Manifest          *mcmanifest.InstanceLaunchManifest
	Username          string
	OfflineUUID       string
	SkinModel         string
	Licensed          *LaunchAuth
	Cosmetics         *LaunchCosmetics
	JoinServerAddress string
	JoinServerPort    int
}

type ClientLaunchReady struct {
	Plan    LaunchPlan
	GameDir string
	Jar     string
	LogPath string
}

func (d *Downloader) PrepareClientLaunch(ctx context.Context, in ClientLaunchInput) (*ClientLaunchReady, error) {
	if in.Manifest == nil {
		return nil, fmt.Errorf("missing manifest")
	}
	username := strings.TrimSpace(in.Username)
	if username == "" {
		username = "Player"
	}
	offlineUUID := strings.TrimSpace(in.OfflineUUID)
	if offlineUUID == "" {
		offlineUUID = "00000000-0000-0000-0000-000000000000"
	}
	skinModel := NormalizeSkinModel(in.SkinModel)
	if in.Licensed != nil {
		username = strings.TrimSpace(in.Licensed.Username)
		offlineUUID = strings.TrimSpace(in.Licensed.UUID)
		if in.Cosmetics == nil || strings.TrimSpace(in.Cosmetics.SkinModel) == "" {
			skinModel = ModelSteve
		}
	}

	jar, libPaths, nativesDir, assetsDir, gameDir, javaBin, err := d.PrepareInstanceGameFiles(ctx, in.Manifest)
	if err != nil {
		return nil, err
	}

	gameUUID := offlineUUID
	if in.Cosmetics != nil && strings.TrimSpace(in.Cosmetics.GameUUID) != "" {
		gameUUID = strings.TrimSpace(in.Cosmetics.GameUUID)
	}

	skinCfg := PlayerSkinConfig{
		UUID:  gameUUID,
		Model: skinModel,
	}
	if in.Cosmetics != nil {
		skinCfg.Model = NormalizeSkinModel(in.Cosmetics.SkinModel)
		if skinCfg.Model == ModelSteve && skinModel != "" {
			skinCfg.Model = skinModel
		}
		if in.Cosmetics.SkinURL != "" {
			if skinPNG, err := DownloadPNG(ctx, d.HTTPClient, in.Cosmetics.SkinURL); err == nil {
				skinCfg.SkinPNG = skinPNG
			}
		}
	}
	if err := EnsurePlayerSkin(gameDir, skinCfg); err != nil {
		return nil, fmt.Errorf("player skin: %w", err)
	}

	quickPlayMultiplayer := JoinServerQuickPlayValue(in.JoinServerAddress, in.JoinServerPort)

	librariesDir, err := d.InstanceLibrariesDir(in.Manifest.InstanceID)
	if err != nil {
		return nil, err
	}
	plan := BuildLaunchPlan(
		in.Manifest,
		jar,
		libPaths,
		nativesDir,
		assetsDir,
		gameDir,
		librariesDir,
		username,
		offlineUUID,
		javaBin,
		in.Licensed,
		quickPlayMultiplayer,
	)

	if in.Cosmetics != nil && in.Cosmetics.UseSkinServer {
		plan.Args = PrependSkinServerJVMArgs(plan.Args, SkinServerConfig{
			Enabled:  true,
			HostBase: in.Cosmetics.SkinServerHost,
		})
	}

	logPath, err := safepath.Join(gameDir, "launch.log")
	if err != nil {
		return nil, err
	}
	if err := writeLaunchDebug(gameDir, plan); err != nil {
		return nil, err
	}
	return &ClientLaunchReady{
		Plan:    plan,
		GameDir: gameDir,
		Jar:     jar,
		LogPath: logPath,
	}, nil
}

func (d *Downloader) PrepareInstanceGameFiles(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (jar string, libPaths []string, nativesDir, assetsDir, gameDir, javaBin string, err error) {
	if manifest == nil {
		return "", nil, "", "", "", "", fmt.Errorf("missing manifest")
	}
	d.progress("prepare", "java runtime")
	javaBin, err = d.EnsureJava(ctx, manifest)
	if err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("java: %w", err)
	}
	if err := d.EnsureLoaderInstalled(ctx, manifest, javaBin); err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("loader install: %w", err)
	}
	d.progress("prepare", "client jar")
	jar, err = d.EnsureClientJar(ctx, manifest)
	if err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("client jar: %w", err)
	}
	d.progress("prepare", "libraries")
	libPaths, err = d.EnsureLibraries(ctx, manifest)
	if err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("libraries: %w", err)
	}
	d.progress("prepare", "natives")
	nativesDir, err = d.EnsureNatives(ctx, manifest)
	if err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("natives: %w", err)
	}
	d.progress("prepare", "assets")
	assetsDir, err = d.EnsureAssets(ctx, manifest)
	if err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("assets: %w", err)
	}

	gameDir, err = d.InstanceGameDir(manifest.InstanceID)
	if err != nil {
		return "", nil, "", "", "", "", err
	}
	if err := EnsureGameLanguage(gameDir, DefaultGameLanguage); err != nil {
		return "", nil, "", "", "", "", fmt.Errorf("game language: %w", err)
	}
	return jar, libPaths, nativesDir, assetsDir, gameDir, javaBin, nil
}

func writeLaunchDebug(gameDir string, plan LaunchPlan) error {
	var b strings.Builder
	b.WriteString(plan.JavaBin)
	for _, arg := range plan.Args {
		b.WriteByte(' ')
		if strings.ContainsAny(arg, " \t\"") {
			b.WriteString(fmt.Sprintf("%q", arg))
		} else {
			b.WriteString(arg)
		}
	}
	b.WriteByte('\n')
	path, err := safepath.Join(gameDir, "launch.cmd.txt")
	if err != nil {
		return err
	}
	return safepath.WriteFileBytes(path, []byte(b.String()), 0o644)
}

func StartClientProcess(ctx context.Context, plan LaunchPlan, logPath string) (*exec.Cmd, error) {
	cmd := proc.CommandContext(ctx, plan.JavaBin, plan.Args...)
	cmd.Dir = plan.WorkingDir
	if logPath != "" {
		if err := safepath.EnsureParent(logPath); err != nil {
			return nil, err
		}
		logFile, err := safepath.Create(logPath)
		if err != nil {
			return nil, err
		}
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
