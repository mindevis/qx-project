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
	"strconv"
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
	ManifestBuilder  ManifestBuilder
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
	dir := d.InstanceVersionsDir(manifest.InstanceID, versionKey)
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

func BuildLaunchPlan(manifest *mcmanifest.InstanceLaunchManifest, clientJar string, libPaths []string, nativesDir, assetsDir, gameDir, librariesDir, username, offlineUUID, javaBin string, licensed *LaunchAuth, quickPlayMultiplayer string) LaunchPlan {
	if gameDir == "" {
		gameDir = filepath.Dir(clientJar)
	}
	if assetsDir == "" {
		assetsDir = filepath.Join(gameDir, "assets")
	}
	if librariesDir == "" {
		librariesDir = filepath.Join(filepath.Dir(clientJar), "libraries")
	}
	subs := launchSubstitutions(manifest, gameDir, assetsDir, librariesDir, nativesDir, username, offlineUUID, licensed, quickPlayMultiplayer)

	var args []string
	if len(manifest.JVMArguments) > 0 {
		args = substituteLaunchArgs(manifest.JVMArguments, subs)
	} else {
		args = []string{"-Xms4G", "-Xmx4G"}
	}
	if nativesDir != "" && !containsArgPrefix(args, "-Djava.library.path=") {
		args = append(args, "-Djava.library.path="+filepath.ToSlash(nativesDir))
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
	args = applyJoinServerArgs(args, quickPlayMultiplayer, manifest)
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

func launchSubstitutions(manifest *mcmanifest.InstanceLaunchManifest, gameDir, assetsDir, librariesDir, nativesDir, username, offlineUUID string, licensed *LaunchAuth, quickPlayMultiplayer string) map[string]string {
	if nativesDir == "" {
		nativesDir = filepath.Join(gameDir, "natives")
	}
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
		"${natives_directory}":      launchPathSlash(nativesDir),
		"${resolution_width}":       "854",
		"${resolution_height}":      "480",
		"${quickPlayPath}":          "",
		"${quickPlaySingleplayer}":  "",
		"${quickPlayMultiplayer}":  quickPlayMultiplayer,
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

func launchArgsContainJoin(args []string) bool {
	for i, arg := range args {
		switch arg {
		case "--quickPlayMultiplayer", "--server":
			if i+1 < len(args) {
				next := strings.TrimSpace(args[i+1])
				if next != "" && !isUnresolvedPlaceholder(next) {
					return true
				}
			}
		}
	}
	return false
}

func splitQuickPlayHostPort(addrPort string) (host, port string) {
	addrPort = strings.TrimSpace(addrPort)
	if addrPort == "" {
		return "", ""
	}
	if strings.HasPrefix(addrPort, "[") {
		if i := strings.LastIndex(addrPort, "]:"); i >= 0 {
			return addrPort[:i+1], addrPort[i+2:]
		}
	}
	if i := strings.LastIndex(addrPort, ":"); i > 0 {
		return addrPort[:i], addrPort[i+1:]
	}
	return addrPort, "25565"
}

func buildJoinServerArgPairs(quickPlayMultiplayer string, manifest *mcmanifest.InstanceLaunchManifest) []string {
	if quickPlayMultiplayer == "" {
		return nil
	}
	mcVersion := ""
	if manifest != nil {
		mcVersion = manifest.MCVersion
	}
	if mcVersionSupportsQuickPlay(mcVersion) {
		return []string{"--quickPlayMultiplayer", quickPlayMultiplayer}
	}
	host, port := splitQuickPlayHostPort(quickPlayMultiplayer)
	if port == "" {
		port = "25565"
	}
	return []string{"--server", host, "--port", port}
}

func applyJoinServerArgs(args []string, quickPlayMultiplayer string, manifest *mcmanifest.InstanceLaunchManifest) []string {
	if quickPlayMultiplayer == "" || launchArgsContainJoin(args) {
		return args
	}
	join := buildJoinServerArgPairs(quickPlayMultiplayer, manifest)
	if len(join) == 0 {
		return args
	}
	for i, arg := range args {
		if arg == "--launchTarget" {
			out := make([]string, 0, len(args)+len(join))
			out = append(out, args[:i]...)
			out = append(out, join...)
			out = append(out, args[i:]...)
			return out
		}
	}
	return append(args, join...)
}

// NormalizeJoinEndpoint splits host:port when address already includes a port.
func NormalizeJoinEndpoint(address string, port int) (host string, normalizedPort int) {
	address = strings.TrimSpace(address)
	if address == "" {
		if port > 0 {
			return "", port
		}
		return "", 25565
	}
	if strings.HasPrefix(address, "[") {
		if i := strings.LastIndex(address, "]:"); i >= 0 {
			hostPart := address[:i+1]
			portPart := strings.TrimSpace(address[i+2:])
			if p, err := strconv.Atoi(portPart); err == nil && p > 0 {
				return hostPart, p
			}
			return hostPart, defaultJoinPort(port)
		}
	}
	if i := strings.LastIndex(address, ":"); i > 0 {
		hostPart := address[:i]
		portPart := strings.TrimSpace(address[i+1:])
		if p, err := strconv.Atoi(portPart); err == nil && p > 0 {
			return hostPart, p
		}
	}
	return address, defaultJoinPort(port)
}

func defaultJoinPort(port int) int {
	if port > 0 {
		return port
	}
	return 25565
}

func JoinServerQuickPlayValue(address string, port int) string {
	host, normalizedPort := NormalizeJoinEndpoint(address, port)
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", host, normalizedPort)
}

func mcVersionSupportsQuickPlay(mcVersion string) bool {
	mcVersion = strings.TrimSpace(mcVersion)
	if mcVersion == "" {
		return false
	}
	parts := strings.Split(mcVersion, ".")
	if len(parts) < 2 {
		return false
	}
	major, errMajor := strconv.Atoi(parts[0])
	minor, errMinor := strconv.Atoi(parts[1])
	if errMajor != nil || errMinor != nil {
		return false
	}
	return major > 1 || (major == 1 && minor >= 20)
}
