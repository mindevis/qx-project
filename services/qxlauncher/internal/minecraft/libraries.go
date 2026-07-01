package minecraft

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
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
	libRoot := d.InstanceLibrariesDir(manifest.InstanceID)
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
		dest := filepath.Join(libRoot, filepath.FromSlash(suffix))
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
	nativesDir := filepath.Join(d.InstanceGameDir(manifest.InstanceID), "natives")
	if err := os.MkdirAll(nativesDir, 0o755); err != nil {
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
	if manifestExpectsNatives(manifest.Libraries, classifier) {
		if extracted == 0 {
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
			jarPath := filepath.Join(d.InstanceCacheDir(instanceID), "natives", nativeCacheName(lib.Name, classifier)+".jar")
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
		jarPath := filepath.Join(d.InstanceCacheDir(instanceID), "natives", nativeCacheName(lib.Name, "")+".jar")
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

func verifyNativeBinariesPresent(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isNativeBinary(e.Name()) {
			return nil
		}
	}
	return fmt.Errorf("no native binaries extracted to %s", dir)
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
		outPath := filepath.Join(destDir, base)
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
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, rc)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
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
