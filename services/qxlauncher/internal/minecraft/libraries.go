package minecraft

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/pkg/safepath"
)

func MavenRelPath(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return ""
	}
	groupParts := strings.Split(strings.ReplaceAll(parts[0], ".", "/"), "/")
	artifact := parts[1]
	version := parts[2]
	suffix := ""
	if len(parts) >= 4 && parts[3] != "" {
		suffix = "-" + parts[3]
	}
	file := fmt.Sprintf("%s-%s%s.jar", artifact, version, suffix)
	elems := append([]string{"libraries"}, groupParts...)
	elems = append(elems, artifact, version, file)
	return filepath.Join(elems...)
}

func (d *Downloader) EnsureLibraries(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) ([]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("missing manifest")
	}
	if manifest.InstanceID == "" {
		return nil, fmt.Errorf("missing instance id")
	}
	libRoot, err := d.InstanceLibrariesDir(manifest.InstanceID)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(manifest.Libraries))
	total := 0
	for _, lib := range manifest.Libraries {
		if isNamedNativeLibrary(lib.Name) {
			continue
		}
		if lib.Downloads == nil || lib.Downloads.Artifact == nil || lib.Downloads.Artifact.URL == "" {
			continue
		}
		total++
	}
	d.progressf("libraries", "checking %d libraries …", total)
	done := 0
	for _, lib := range manifest.Libraries {
		if isNamedNativeLibrary(lib.Name) {
			continue
		}
		if lib.Downloads == nil || lib.Downloads.Artifact == nil || lib.Downloads.Artifact.URL == "" {
			continue
		}
		rel := MavenRelPath(lib.Name)
		if rel == "" {
			continue
		}
		suffix := strings.TrimPrefix(filepath.ToSlash(rel), "libraries/")
		dest, err := safepath.JoinRel(libRoot, suffix)
		if err != nil {
			return nil, fmt.Errorf("library %s: %w", lib.Name, err)
		}
		if err := d.downloadIfNeeded(ctx, lib.Downloads.Artifact.URL, dest, lib.Downloads.Artifact.Sha1); err != nil {
			return nil, fmt.Errorf("library %s: %w", lib.Name, err)
		}
		paths = append(paths, dest)
		done++
		if done == 1 || done%10 == 0 || done == total {
			d.progressf("libraries", "%d/%d", done, total)
		}
	}
	return paths, nil
}

func (d *Downloader) EnsureNatives(ctx context.Context, manifest *mcmanifest.InstanceLaunchManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("missing manifest")
	}
	gameDir, err := d.InstanceGameDir(manifest.InstanceID)
	if err != nil {
		return "", err
	}
	nativesDir, err := safepath.Join(gameDir, "natives")
	if err != nil {
		return "", err
	}
	if err := safepath.EnsureDir(nativesDir); err != nil {
		return "", err
	}
	classifier := nativeClassifier()
	extracted := 0
	for _, lib := range manifest.Libraries {
		n, err := d.ensureLibraryNatives(ctx, manifest.InstanceID, lib, classifier, nativesDir)
		if err != nil {
			return "", fmt.Errorf("library %s: %w", lib.Name, err)
		}
		extracted += n
	}
	if err := ensureNativeRuntimeLayout(nativesDir); err != nil {
		return "", err
	}
	if manifestExpectsNatives(manifest.Libraries, classifier) {
		if extracted == 0 && nativeLibrarySearchDir(nativesDir) == "" {
			return "", fmt.Errorf("no %s native libraries found in manifest", classifier)
		}
		if err := verifyNativeBinariesPresent(nativesDir); err != nil {
			return "", err
		}
	}
	return nativesDir, nil
}

func manifestExpectsNatives(libs []mcmanifest.Library, classifier string) bool {
	for _, lib := range libs {
		if isNamedNativeLibraryForClassifier(lib.Name, classifier) {
			return true
		}
		if lib.Downloads != nil && lib.Downloads.Classifiers != nil {
			if artifact, ok := lib.Downloads.Classifiers[classifier]; ok && artifact.URL != "" {
				return true
			}
		}
	}
	return false
}

func (d *Downloader) ensureLibraryNatives(ctx context.Context, instanceID string, lib mcmanifest.Library, classifier, nativesDir string) (int, error) {
	extracted := 0
	if lib.Downloads != nil && lib.Downloads.Classifiers != nil {
		if artifact, ok := lib.Downloads.Classifiers[classifier]; ok && artifact.URL != "" {
			cacheDir, err := d.InstanceCacheDir(instanceID)
			if err != nil {
				return 0, err
			}
			jarPath, err := safepath.Join(cacheDir, "natives", nativeCacheName(lib.Name, classifier)+".jar")
			if err != nil {
				return 0, err
			}
			if err := d.downloadIfNeeded(ctx, artifact.URL, jarPath, artifact.Sha1); err != nil {
				return 0, err
			}
			n, err := extractNativeBinaries(jarPath, nativesDir)
			if err != nil {
				return 0, err
			}
			extracted += n
		}
	}
	if isNamedNativeLibraryForClassifier(lib.Name, classifier) {
		if lib.Downloads == nil || lib.Downloads.Artifact == nil || lib.Downloads.Artifact.URL == "" {
			return 0, fmt.Errorf("missing native artifact download")
		}
		cacheDir, err := d.InstanceCacheDir(instanceID)
		if err != nil {
			return 0, err
		}
		jarPath, err := safepath.Join(cacheDir, "natives", nativeCacheName(lib.Name, "")+".jar")
		if err != nil {
			return 0, err
		}
		if err := d.downloadIfNeeded(ctx, lib.Downloads.Artifact.URL, jarPath, lib.Downloads.Artifact.Sha1); err != nil {
			return 0, err
		}
		n, err := extractNativeBinaries(jarPath, nativesDir)
		if err != nil {
			return 0, err
		}
		extracted += n
	}
	return extracted, nil
}

func nativeCacheName(libName, classifier string) string {
	if classifier != "" {
		return strings.ReplaceAll(libName, ":", "_") + "-" + classifier
	}
	return strings.ReplaceAll(libName, ":", "_")
}

func nativeClassifier() string {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "arm64":
			return "natives-windows-arm64"
		case "386":
			return "natives-windows-x86"
		default:
			return "natives-windows"
		}
	case "darwin":
		switch runtime.GOARCH {
		case "arm64":
			return "natives-macos-arm64"
		default:
			return "natives-macos"
		}
	default:
		switch runtime.GOARCH {
		case "arm64":
			return "natives-linux-arm64"
		case "arm":
			return "natives-linux-arm32"
		default:
			return "natives-linux"
		}
	}
}

func isNamedNativeLibrary(name string) bool {
	parts := strings.Split(name, ":")
	return len(parts) >= 4 && strings.HasPrefix(parts[3], "natives-")
}

func isNamedNativeLibraryForClassifier(name, classifier string) bool {
	parts := strings.Split(name, ":")
	return len(parts) >= 4 && parts[3] == classifier
}

func nativeRuntimeSubdirs() []string {
	return []string{"java", "jna", "lwjgl", "netty"}
}

func dirHasNativeBinary(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && isNativeBinary(e.Name()) {
			return true
		}
	}
	return false
}

func nativeLibrarySearchDir(nativesDir string) string {
	if nativesDir == "" {
		return ""
	}
	for _, candidate := range []string{
		filepath.Join(nativesDir, "java"),
		filepath.Join(nativesDir, "lwjgl"),
		nativesDir,
	} {
		if dirHasNativeBinary(candidate) {
			return candidate
		}
	}
	return ""
}

func verifyNativeBinariesPresent(dir string) error {
	if nativeLibrarySearchDir(dir) != "" {
		return nil
	}
	return fmt.Errorf("no native binaries extracted to %s", dir)
}

func collectNativeBinaries(nativesDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	dirs := []string{nativesDir}
	for _, name := range nativeRuntimeSubdirs() {
		dirs = append(dirs, filepath.Join(nativesDir, name))
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !isNativeBinary(e.Name()) {
				continue
			}
			if _, ok := files[e.Name()]; ok {
				continue
			}
			src, err := safepath.Join(dir, e.Name())
			if err != nil {
				return nil, err
			}
			data, err := safepath.ReadFileBytes(src)
			if err != nil {
				return nil, err
			}
			files[e.Name()] = data
		}
	}
	return files, nil
}

// Minecraft 26.2+ looks for natives in subfolders:
// ${natives_directory}/java, /lwjgl, /jna, /netty — not only the natives root.
func ensureNativeRuntimeLayout(nativesDir string) error {
	for _, name := range nativeRuntimeSubdirs() {
		dir, err := safepath.Join(nativesDir, name)
		if err != nil {
			return err
		}
		if err := safepath.EnsureDir(dir); err != nil {
			return err
		}
	}
	files, err := collectNativeBinaries(nativesDir)
	if err != nil {
		return err
	}
	destDirs := []string{nativesDir}
	for _, name := range nativeRuntimeSubdirs() {
		dir, err := safepath.Join(nativesDir, name)
		if err != nil {
			return err
		}
		destDirs = append(destDirs, dir)
	}
	for name, data := range files {
		for _, destDir := range destDirs {
			dest, err := safepath.Join(destDir, name)
			if err != nil {
				return err
			}
			if err := safepath.WriteFileBytes(dest, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractNativeBinaries(jarPath, destDir string) (int, error) {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return 0, err
	}
	defer r.Close()
	extracted := 0
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base, err := safepath.ZipEntryBase(f.Name)
		if err != nil {
			continue
		}
		if !isNativeBinary(base) {
			continue
		}
		outPath, err := safepath.Join(destDir, base)
		if err != nil {
			continue
		}
		if err := extractZipFile(f, outPath); err != nil {
			return extracted, err
		}
		extracted++
	}
	return extracted, nil
}

func isNativeBinary(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".dll"):
		return true
	case strings.HasSuffix(lower, ".so"):
		return true
	case strings.HasSuffix(lower, ".dylib"):
		return true
	default:
		return false
	}
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	if err := safepath.EnsureParent(dest); err != nil {
		return err
	}
	return safepath.WriteStreamAtomic(dest, rc)
}

func BuildClasspath(entries []string) string {
	sep := string(os.PathListSeparator)
	return strings.Join(entries, sep)
}

func buildLegacyClassPath(libPaths []string) string {
	sep := string(os.PathListSeparator)
	entries := make([]string, 0, len(libPaths))
	seen := make(map[string]struct{}, len(libPaths))
	for _, p := range libPaths {
		p = filepath.Clean(p)
		if p == "" || excludedFromLegacyClasspath(p) {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		entries = append(entries, filepath.ToSlash(p))
	}
	return strings.Join(entries, sep)
}

func excludedFromLegacyClasspath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(base, "-universal.jar"):
		return true
	case strings.HasSuffix(base, "-client.jar") && (strings.Contains(base, "neoforge") || strings.Contains(base, "forge")):
		return true
	case strings.Contains(strings.ToLower(path), "net/minecraft/client") && strings.Contains(base, "client-"):
		return true
	default:
		return false
	}
}
