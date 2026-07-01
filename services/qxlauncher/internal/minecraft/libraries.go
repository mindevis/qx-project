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
		if lib.Downloads == nil || lib.Downloads.Artifact == nil || lib.Downloads.Artifact.URL == "" {
			continue
		}
		total++
	}
	d.progressf("libraries", "checking %d libraries …", total)
	done := 0
	for _, lib := range manifest.Libraries {
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
	for _, lib := range manifest.Libraries {
		if lib.Downloads == nil || lib.Downloads.Classifiers == nil {
			continue
		}
		artifact, ok := lib.Downloads.Classifiers[classifier]
		if !ok || artifact.URL == "" {
			continue
		}
		cacheName := strings.ReplaceAll(lib.Name, ":", "_") + "-" + classifier + ".jar"
		jarPath := filepath.Join(d.InstanceCacheDir(manifest.InstanceID), "natives", cacheName)
		if err := d.downloadIfNeeded(ctx, artifact.URL, jarPath, artifact.Sha1); err != nil {
			return "", err
		}
		if err := extractNativeBinaries(jarPath, nativesDir); err != nil {
			return "", err
		}
	}
	return nativesDir, nil
}

func nativeClassifier() string {
	switch runtime.GOOS {
	case "windows":
		return "natives-windows"
	case "darwin":
		return "natives-macos"
	default:
		return "natives-linux"
	}
}

func extractNativeBinaries(jarPath, destDir string) error {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer r.Close()
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
			return err
		}
	}
	return nil
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
