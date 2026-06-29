package minecraft

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type Downloader struct {
	RootDir          string
	JavaPath         string
	SkipJavaDownload bool
	AssetsCDN        string
	HTTPClient       *http.Client
	OnProgress       ProgressFunc
}

func NewDownloader(root string) *Downloader {
	return &Downloader{
		RootDir:    root,
		HTTPClient: defaultHTTPClient(),
	}
}

func (d *Downloader) EnsureClientJar(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if manifest == nil || manifest.ClientJar.URL == "" {
		return "", fmt.Errorf("missing client jar")
	}
	versionKey := manifest.MCVersion
	jarName := manifest.MCVersion + ".jar"
	if manifest.VersionID != "" {
		versionKey = manifest.VersionID
		jarName = manifest.VersionID + ".jar"
	}
	dir := filepath.Join(d.RootDir, "instances", manifest.InstanceID, "versions", versionKey)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, jarName)
	d.progressf("client", "downloading %s …", jarName)
	return dest, d.downloadIfNeeded(ctx, manifest.ClientJar.URL, dest, manifest.ClientJar.Sha1)
}

func (d *Downloader) downloadIfNeeded(ctx context.Context, url, dest, sha1hex string) error {
	if sha1hex != "" {
		if b, err := os.ReadFile(dest); err == nil {
			if hex.EncodeToString(sha1Sum(b)) == strings.ToLower(sha1hex) {
				return nil
			}
		}
	}
	return d.downloadWithRetry(ctx, url, dest)
}

func sha1Sum(b []byte) []byte {
	sum := sha1.Sum(b)
	return sum[:]
}

type LaunchPlan struct {
	JavaBin    string
	Args       []string
	MainClass  string
	WorkingDir string
}

func BuildLaunchPlan(manifest *mcmanifest.InstanceLaunchManifest, clientJar string, libPaths []string, nativesDir, assetsDir, gameDir, librariesDir, username, offlineUUID, javaBin string, licensed *LaunchAuth) LaunchPlan {
	if gameDir == "" {
		gameDir = filepath.Dir(clientJar)
	}
	if assetsDir == "" {
		assetsDir = filepath.Join(gameDir, "assets")
	}
	if librariesDir == "" {
		librariesDir = filepath.Join(filepath.Dir(clientJar), "libraries")
	}
	subs := launchSubstitutions(manifest, gameDir, assetsDir, librariesDir, username, offlineUUID, licensed)

	var args []string
	if len(manifest.JVMArguments) > 0 {
		args = substituteLaunchArgs(manifest.JVMArguments, subs)
	} else {
		args = []string{"-Xmx2G"}
	}
	if nativesDir != "" && !containsArgPrefix(args, "-Djava.library.path=") {
		args = append(args, "-Djava.library.path="+nativesDir)
	}

	if !usesModulePathLaunch(manifest) {
		cpEntries := append(append([]string{}, libPaths...), clientJar)
		args = append(args, "-cp", BuildClasspath(cpEntries))
	} else if !containsArgPrefix(args, "-DlegacyClassPath=") {
		args = append(args, "-DlegacyClassPath="+buildLegacyClassPath(libPaths))
	}
	args = append(args, manifest.MainClass)

	if len(manifest.GameArguments) > 0 {
		args = append(args, sanitizeLaunchArgs(substituteLaunchArgs(manifest.GameArguments, subs))...)
	} else {
		gameUUID := offlineUUID
		accessToken := "0"
		userType := "legacy"
		if licensed != nil {
			gameUUID = strings.ReplaceAll(licensed.UUID, "-", "")
			accessToken = licensed.AccessToken
			userType = "msa"
		}
		args = append(args,
			"--username", username,
			"--version", manifest.MCVersion,
			"--gameDir", gameDir,
			"--assetsDir", assetsDir,
			"--assetIndex", manifest.AssetIndex.ID,
			"--uuid", gameUUID,
			"--accessToken", accessToken,
			"--userType", userType,
		)
	}
	if javaBin == "" {
		javaBin = ResolveJavaBin("")
	}
	return LaunchPlan{
		JavaBin:    javaBin,
		Args:       args,
		MainClass:  manifest.MainClass,
		WorkingDir: gameDir,
	}
}

func usesModulePathLaunch(manifest *mcmanifest.InstanceLaunchManifest) bool {
	if manifest == nil {
		return false
	}
	switch manifest.Loader {
	case mcmanifest.LoaderForge, mcmanifest.LoaderNeoForge:
		return true
	}
	for _, arg := range manifest.JVMArguments {
		if arg == "-p" {
			return true
		}
	}
	return false
}

func containsArgPrefix(args []string, prefix string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

func launchPathSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func launchSubstitutions(manifest *mcmanifest.InstanceLaunchManifest, gameDir, assetsDir, librariesDir, username, offlineUUID string, licensed *LaunchAuth) map[string]string {
	versionName := ""
	assetIndex := ""
	if manifest != nil {
		versionName = manifest.MCVersion
		if manifest.VersionID != "" {
			versionName = manifest.VersionID
		}
		assetIndex = manifest.AssetIndex.ID
	}
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	authUUID := offlineUUID
	authToken := "0"
	userType := "legacy"
	if licensed != nil {
		authUUID = strings.ReplaceAll(licensed.UUID, "-", "")
		authToken = licensed.AccessToken
		userType = "msa"
		if username == "" {
			username = licensed.Username
		}
	}
	return map[string]string{
		"${auth_player_name}":       username,
		"${version_name}":           versionName,
		"${game_directory}":         launchPathSlash(gameDir),
		"${assets_root}":            launchPathSlash(assetsDir),
		"${assets_index_name}":      assetIndex,
		"${auth_uuid}":              authUUID,
		"${auth_access_token}":      authToken,
		"${user_type}":              userType,
		"${version_type}":           "release",
		"${clientid}":               "",
		"${auth_xuid}":              "",
		"${library_directory}":      launchPathSlash(librariesDir),
		"${classpath_separator}":    sep,
		"${natives_directory}":      launchPathSlash(filepath.Join(gameDir, "natives")),
		"${resolution_width}":       "854",
		"${resolution_height}":      "480",
		"${quickPlayPath}":          "",
		"${quickPlaySingleplayer}":  "",
		"${quickPlayMultiplayer}":  "",
		"${quickPlayRealms}":        "",
	}
}

func substituteLaunchArgs(items []string, subs map[string]string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, substituteLaunchArg(item, subs))
	}
	return out
}

func substituteLaunchArg(item string, subs map[string]string) string {
	if strings.HasPrefix(item, "${") {
		if v, ok := subs[item]; ok {
			return v
		}
	}
	for k, v := range subs {
		item = strings.ReplaceAll(item, k, v)
	}
	return item
}

func sanitizeLaunchArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" || isUnresolvedPlaceholder(arg) {
			continue
		}
		if isOptionalValueFlag(arg) {
			if i+1 >= len(args) {
				continue
			}
			next := args[i+1]
			if next == "" || isUnresolvedPlaceholder(next) {
				i++
				continue
			}
		}
		out = append(out, arg)
		if isOptionalValueFlag(arg) {
			i++
			out = append(out, args[i])
		}
	}
	return out
}

func isUnresolvedPlaceholder(s string) bool {
	return strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}")
}

func isOptionalValueFlag(arg string) bool {
	switch arg {
	case "--clientId", "--xuid",
		"--quickPlayPath", "--quickPlaySingleplayer", "--quickPlayMultiplayer", "--quickPlayRealms":
		return true
	default:
		return false
	}
}
