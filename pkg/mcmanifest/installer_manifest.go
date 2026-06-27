package mcmanifest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	forgeMavenBase    = "https://maven.minecraftforge.net/net/minecraftforge/forge"
	neoforgeMavenBase = "https://maven.neoforged.net/releases/net/neoforged/neoforge"
)

func forgeInstallerURL(mcVersion, loaderVersion string) string {
	artifact := fmt.Sprintf("%s-%s", mcVersion, loaderVersion)
	return fmt.Sprintf("%s/%s/forge-%s-installer.jar", forgeMavenBase, artifact, artifact)
}

func neoforgeInstallerURL(loaderVersion string) string {
	loaderVersion = strings.TrimSpace(loaderVersion)
	return fmt.Sprintf("%s/%s/neoforge-%s-installer.jar", neoforgeMavenBase, loaderVersion, loaderVersion)
}

func (c *Client) buildInstallerManifest(ctx context.Context, instanceID, name, mcVersion, loader, loaderVersion, installerURL string) (*InstanceLaunchManifest, error) {
	body, err := c.get(ctx, installerURL)
	if err != nil {
		return nil, fmt.Errorf("download forge installer: %w", err)
	}
	meta, err := parseVersionMetaFromInstaller(body)
	if err != nil {
		return nil, err
	}
	meta, err = c.resolveInheritedMeta(ctx, meta)
	if err != nil {
		return nil, err
	}
	out := launchManifestFromMeta(instanceID, name, mcVersion, loader, loaderVersion, installerURL, meta)
	if artifact, err := clientArtifactFromInstaller(body); err == nil {
		out.LoaderClientJar = artifact
	}
	return out, nil
}

func (c *Client) fetchVersionMetaFromInstaller(ctx context.Context, installerURL string) (*VersionMeta, error) {
	body, err := c.get(ctx, installerURL)
	if err != nil {
		return nil, fmt.Errorf("download forge installer: %w", err)
	}
	return parseVersionMetaFromInstaller(body)
}

func parseVersionMetaFromInstaller(body []byte) (*VersionMeta, error) {
	raw, err := readZipEntry(body, "version.json")
	if err != nil {
		return nil, err
	}
	var meta VersionMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("parse installer version.json: %w", err)
	}
	return &meta, nil
}

func clientArtifactFromInstaller(body []byte) (LoaderGeneratedJar, error) {
	raw, err := readZipEntry(body, "install_profile.json")
	if err != nil {
		return LoaderGeneratedJar{}, err
	}
	var profile struct {
		Data struct {
			PATCHED struct {
				Client string `json:"client"`
			} `json:"PATCHED"`
			PATCHED_SHA struct {
				Client string `json:"client"`
			} `json:"PATCHED_SHA"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		return LoaderGeneratedJar{}, fmt.Errorf("parse install_profile.json: %w", err)
	}
	name := parseBracketMavenCoord(profile.Data.PATCHED.Client)
	if name == "" {
		return LoaderGeneratedJar{}, fmt.Errorf("missing PATCHED client in install_profile.json")
	}
	rel := libraryRelPath(name)
	if rel == "" {
		return LoaderGeneratedJar{}, fmt.Errorf("invalid PATCHED client coord %q", name)
	}
	sha1hex := strings.Trim(profile.Data.PATCHED_SHA.Client, "'\"")
	return LoaderGeneratedJar{RelativePath: rel, Sha1: sha1hex}, nil
}

func parseBracketMavenCoord(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

// DefaultLoaderClientJar returns the expected installer-generated client jar path for Forge/NeoForge.
func DefaultLoaderClientJar(loader, mcVersion, loaderVersion string) LoaderGeneratedJar {
	loader = NormalizeLoader(loader)
	loaderVersion = strings.TrimSpace(loaderVersion)
	mcVersion = strings.TrimSpace(mcVersion)
	switch loader {
	case LoaderForge:
		if mcVersion == "" || loaderVersion == "" {
			return LoaderGeneratedJar{}
		}
		artifact := fmt.Sprintf("%s-%s", mcVersion, loaderVersion)
		return LoaderGeneratedJar{RelativePath: libraryRelPath("net.minecraftforge:forge:" + artifact + ":client")}
	case LoaderNeoForge:
		if loaderVersion == "" {
			return LoaderGeneratedJar{}
		}
		return LoaderGeneratedJar{RelativePath: libraryRelPath("net.neoforged:neoforge:" + loaderVersion + ":client")}
	default:
		return LoaderGeneratedJar{}
	}
}

func libraryRelPath(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return ""
	}
	groupPath := strings.ReplaceAll(parts[0], ".", "/")
	artifact := parts[1]
	version := parts[2]
	suffix := ""
	if len(parts) >= 4 && parts[3] != "" {
		suffix = "-" + parts[3]
	}
	file := fmt.Sprintf("%s-%s%s.jar", artifact, version, suffix)
	return strings.Join(append([]string{"libraries"}, strings.Split(groupPath, "/")...), "/") + "/" + artifact + "/" + version + "/" + file
}

func readZipEntry(zipBytes []byte, name string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("open installer archive: %w", err)
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		_, readErr := buf.ReadFrom(rc)
		closeErr := rc.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("%q not found in installer", name)
}
