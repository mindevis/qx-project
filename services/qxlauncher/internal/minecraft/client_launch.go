package minecraft

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
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
	Manifest    *mcmanifest.InstanceLaunchManifest
	Username    string
	OfflineUUID string
	SkinModel   string
	Licensed    *LaunchAuth
	Cosmetics   *LaunchCosmetics
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

	d.progress("prepare", "java runtime")
	javaBin, err := d.EnsureJava(ctx, in.Manifest)
	if err != nil {
		return nil, fmt.Errorf("java: %w", err)
	}
	if err := d.EnsureLoaderInstalled(ctx, in.Manifest, javaBin); err != nil {
		return nil, fmt.Errorf("loader install: %w", err)
	}
	d.progress("prepare", "client jar")
	jar, err := d.EnsureClientJar(ctx, in.Manifest)
	if err != nil {
		return nil, fmt.Errorf("client jar: %w", err)
	}
	d.progress("prepare", "libraries")
	libPaths, err := d.EnsureLibraries(ctx, in.Manifest)
	if err != nil {
		return nil, fmt.Errorf("libraries: %w", err)
	}
	d.progress("prepare", "natives")
	nativesDir, err := d.EnsureNatives(ctx, in.Manifest)
	if err != nil {
		nativesDir = ""
	}
	d.progress("prepare", "assets")
	assetsDir, err := d.EnsureAssets(ctx, in.Manifest)
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}

	gameDir := d.InstanceGameDir(in.Manifest.InstanceID)
	if err := EnsureGameLanguage(gameDir, DefaultGameLanguage); err != nil {
		return nil, fmt.Errorf("game language: %w", err)
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

	plan := BuildLaunchPlan(
		in.Manifest,
		jar,
		libPaths,
		nativesDir,
		assetsDir,
		gameDir,
		filepath.Join(d.RootDir, "libraries"),
		username,
		offlineUUID,
		javaBin,
		in.Licensed,
	)

	if in.Cosmetics != nil && in.Cosmetics.UseSkinServer {
		plan.Args = PrependSkinServerJVMArgs(plan.Args, SkinServerConfig{
			Enabled:  true,
			HostBase: in.Cosmetics.SkinServerHost,
		})
	}

	logPath := filepath.Join(gameDir, "launch.log")
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
	return os.WriteFile(filepath.Join(gameDir, "launch.cmd.txt"), []byte(b.String()), 0o644)
}

func StartClientProcess(ctx context.Context, plan LaunchPlan, logPath string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, plan.JavaBin, plan.Args...)
	cmd.Dir = plan.WorkingDir
	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return nil, err
		}
		logFile, err := os.Create(logPath)
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
