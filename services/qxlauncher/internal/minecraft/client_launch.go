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

type ClientLaunchInput struct {
	Manifest    *mcmanifest.InstanceLaunchManifest
	Username    string
	OfflineUUID string
	SkinModel   string
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
	if err := EnsureOfflineSkin(gameDir, offlineUUID, skinModel); err != nil {
		return nil, fmt.Errorf("offline skin: %w", err)
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
	)
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
