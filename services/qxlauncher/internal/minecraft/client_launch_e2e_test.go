package minecraft

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type clientLaunchE2EOptions struct {
	EnvFlag         string
	MCVersion       string
	LoaderVersion         string
	LoaderVersionFallback string
	Loader                string
	InstanceID      string
	DataDirEnv      string
	DataDirDefault  string
	UsernameEnv     string
	UsernameDefault string
	AliveEnv        string
	KeepEnv         string
	ResolveLatest   func(context.Context, string) (string, error)
	LogReady        func(string) bool
	LoaderLabel     string
}

func runClientLaunchE2E(t *testing.T, opts clientLaunchE2EOptions) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipped in short mode")
	}
	if os.Getenv(opts.EnvFlag) != "1" {
		t.Skipf("set %s=1 to run full %s client launch", opts.EnvFlag, opts.LoaderLabel)
	}

	mcVersion := envDefault("QX_"+strings.ToUpper(opts.Loader)+"_MC_VERSION", opts.MCVersion)
	loaderVersion := envDefault("QX_"+strings.ToUpper(opts.Loader)+"_LOADER_VERSION", opts.LoaderVersion)
	username := envDefault(opts.UsernameEnv, opts.UsernameDefault)
	dataDir := envDefault(opts.DataDirEnv, filepath.Join(os.TempDir(), opts.DataDirDefault))
	aliveFor := envDuration(opts.AliveEnv, 45*time.Second)
	keepOpen := os.Getenv(opts.KeepEnv) == "1"

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	if loaderVersion == "" && opts.ResolveLatest != nil {
		var err error
		loaderVersion, err = opts.ResolveLatest(ctx, mcVersion)
		if err != nil {
			if opts.LoaderVersionFallback != "" {
				t.Logf("resolve latest failed (%v), using fallback %s", err, opts.LoaderVersionFallback)
				loaderVersion = opts.LoaderVersionFallback
			} else {
				t.Fatalf("resolve latest %s: %v", opts.LoaderLabel, err)
			}
		} else {
			t.Logf("resolved latest %s: %s", opts.LoaderLabel, loaderVersion)
		}
	} else if loaderVersion == "" && opts.LoaderVersionFallback != "" {
		loaderVersion = opts.LoaderVersionFallback
	}

	t.Logf("building %s manifest %s / %s …", opts.LoaderLabel, mcVersion, loaderVersion)
	manifest, err := mcmanifest.NewClient().BuildInstanceManifest(
		ctx,
		opts.InstanceID,
		opts.LoaderLabel+" E2E",
		mcVersion,
		opts.Loader,
		loaderVersion,
	)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	t.Logf("manifest version_id=%s main_class=%s libraries=%d", manifest.VersionID, manifest.MainClass, len(manifest.Libraries))
	t.Logf("data dir: %s (downloads are cached between runs)", dataDir)
	t.Log("preparing downloads — first run can take 10–20 min, progress below:")

	dl := NewDownloader(dataDir)
	dl.OnProgress = func(phase, message string) {
		t.Logf("[%s] %s", phase, message)
	}
	ready, err := dl.PrepareClientLaunch(ctx, ClientLaunchInput{
		Manifest:    manifest,
		Username:    username,
		OfflineUUID: "00000000-0000-0000-0000-000000000001",
		SkinModel:   ModelSteve,
	})
	if err != nil {
		t.Fatalf("prepare launch: %v", err)
	}

	t.Logf("game dir: %s", ready.GameDir)
	t.Logf("launch log: %s", ready.LogPath)
	t.Logf("command file: %s", filepath.Join(ready.GameDir, "launch.cmd.txt"))
	t.Logf("java: %s", ready.Plan.JavaBin)
	t.Logf("main: %s", ready.Plan.MainClass)

	cmd, err := StartClientProcess(context.Background(), ready.Plan, ready.LogPath)
	if err != nil {
		t.Fatalf("start process: %v", err)
	}
	t.Logf("%s client started pid=%d — check the Minecraft window", opts.LoaderLabel, cmd.Process.Pid)

	exited := make(chan error, 1)
	go func() {
		exited <- cmd.Wait()
	}()

	if keepOpen {
		t.Logf("%s=1: waiting until you close Minecraft …", opts.KeepEnv)
		if err := <-exited; err != nil {
			dumpLaunchLog(t, ready.LogPath)
			t.Fatalf("%s client exited: %v", strings.ToLower(opts.LoaderLabel), err)
		}
		t.Logf("%s client closed normally", opts.LoaderLabel)
		return
	}

	t.Logf("waiting %s to confirm the client stays alive …", aliveFor)
	select {
	case err := <-exited:
		dumpLaunchLog(t, ready.LogPath)
		if err != nil {
			t.Fatalf("%s client exited too early: %v", strings.ToLower(opts.LoaderLabel), err)
		}
		if opts.LogReady != nil && opts.LogReady(ready.LogPath) {
			t.Logf("%s client exited cleanly after successful initialization", opts.LoaderLabel)
			return
		}
		t.Fatalf("%s client exited too early with code 0", strings.ToLower(opts.LoaderLabel))
	case <-time.After(aliveFor):
		t.Logf("%s client still running after %s — success", opts.LoaderLabel, aliveFor)
		_ = cmd.Process.Kill()
		<-exited
	}
}

func envDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func dumpLaunchLog(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Logf("launch log missing (%s): %v", path, err)
		return
	}
	const tail = 8000
	out := string(b)
	if len(out) > tail {
		out = out[len(out)-tail:]
	}
	t.Logf("--- launch.log tail ---\n%s\n--- end ---", out)
}

func launchLogShowsModLoaderReady(path string, extraMarkers ...string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	log := string(b)
	markers := append([]string{
		"Launching target 'forgeclient'",
		"Created:",
	}, extraMarkers...)
	for _, marker := range markers {
		if !strings.Contains(log, marker) {
			return false
		}
	}
	return true
}

func launchLogContains(path string, markers ...string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	log := string(b)
	for _, marker := range markers {
		if !strings.Contains(log, marker) {
			return false
		}
	}
	return true
}
