package mcmanifest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const DefaultManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest_v2.json"

type Client struct {
	ManifestURL string
	HTTPClient  *http.Client
}

func NewClient() *Client {
	return &Client{
		ManifestURL: DefaultManifestURL,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

type VersionEntry struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	URL             string `json:"url"`
	Time            string `json:"time"`
	ReleaseTime     string `json:"releaseTime"`
	Sha1            string `json:"sha1"`
	ComplianceLevel int    `json:"complianceLevel"`
}

type VersionManifestV2 struct {
	Latest   map[string]string `json:"latest"`
	Versions []VersionEntry    `json:"versions"`
}

type VersionMeta struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	MainClass    string            `json:"mainClass"`
	Assets       string            `json:"assets"`
	InheritsFrom string            `json:"inheritsFrom,omitempty"`
	AssetIndex   AssetIndexRef     `json:"assetIndex"`
	Downloads    Downloads         `json:"downloads"`
	Libraries    []Library         `json:"libraries"`
	JavaVersion  JavaVersion       `json:"javaVersion"`
	Arguments    *VersionArguments `json:"arguments,omitempty"`
}

type AssetIndexRef struct {
	ID        string `json:"id"`
	Sha1      string `json:"sha1"`
	Size      int64  `json:"size"`
	TotalSize int64  `json:"totalSize"`
	URL       string `json:"url"`
}

type Downloads struct {
	Client DownloadFile `json:"client"`
}

type DownloadFile struct {
	Sha1 string `json:"sha1"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type Library struct {
	Name      string            `json:"name"`
	Downloads *LibraryDownloads `json:"downloads,omitempty"`
	Rules     []Rule            `json:"rules,omitempty"`
	RepoURL   string            `json:"url,omitempty"`
	Sha1      string            `json:"sha1,omitempty"`
}

type LibraryDownloads struct {
	Artifact    *DownloadFile `json:"artifact,omitempty"`
	Classifiers map[string]DownloadFile `json:"classifiers,omitempty"`
}

type Rule struct {
	Action   string                     `json:"action"`
	OS       *RuleOS                    `json:"os,omitempty"`
	Features map[string]json.RawMessage `json:"features,omitempty"`
}

type RuleOS struct {
	Name string `json:"name"`
}

type JavaVersion struct {
	Component    string `json:"component"`
	MajorVersion int    `json:"majorVersion"`
}

type InstanceLaunchManifest struct {
	InstanceID    string        `json:"instance_id"`
	Name          string        `json:"name"`
	MCVersion     string        `json:"mc_version"`
	Loader          string        `json:"loader"`
	LoaderVersion   string        `json:"loader_version,omitempty"`
	VersionID       string        `json:"version_id,omitempty"`
	VersionURL    string        `json:"version_url"`
	MainClass     string        `json:"main_class"`
	AssetIndex    AssetIndexRef `json:"asset_index"`
	ClientJar     DownloadFile  `json:"client_jar"`
	Libraries     []Library     `json:"libraries"`
	JavaMajor       int      `json:"java_major"`
	JavaComponent   string   `json:"java_component,omitempty"`
	GameArguments    []string           `json:"game_arguments,omitempty"`
	JVMArguments     []string           `json:"jvm_arguments,omitempty"`
	LoaderClientJar  LoaderGeneratedJar `json:"loader_client_jar,omitempty"`
}

// LoaderGeneratedJar is produced locally by Forge/NeoForge installer processors (not on Maven).
type LoaderGeneratedJar struct {
	RelativePath string `json:"relative_path,omitempty"`
	Sha1         string `json:"sha1,omitempty"`
}

type McVersionItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type McVersionsList struct {
	Latest map[string]string `json:"latest"`
	Items  []McVersionItem   `json:"items"`
}

func (c *Client) ListVersions(ctx context.Context) (*McVersionsList, error) {
	manifest, err := c.FetchVersionManifest(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]McVersionItem, 0, len(manifest.Versions))
	for _, v := range manifest.Versions {
		items = append(items, McVersionItem{ID: v.ID, Type: v.Type})
	}
	return &McVersionsList{
		Latest: manifest.Latest,
		Items:  items,
	}, nil
}

func (c *Client) FetchVersionManifest(ctx context.Context) (*VersionManifestV2, error) {
	body, err := c.get(ctx, c.ManifestURL)
	if err != nil {
		return nil, err
	}
	var manifest VersionManifestV2
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("parse version manifest: %w", err)
	}
	return &manifest, nil
}

func (c *Client) ResolveVersionURL(ctx context.Context, versionID string) (string, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return "", fmt.Errorf("empty version id")
	}
	manifest, err := c.FetchVersionManifest(ctx)
	if err != nil {
		return "", err
	}
	for _, v := range manifest.Versions {
		if v.ID == versionID {
			return v.URL, nil
		}
	}
	if url, ok := manifest.Latest[versionID]; ok && strings.HasPrefix(url, "http") {
		return url, nil
	}
	return "", fmt.Errorf("version %q not found", versionID)
}

func (c *Client) FetchVersionMeta(ctx context.Context, versionURL string) (*VersionMeta, error) {
	body, err := c.get(ctx, versionURL)
	if err != nil {
		return nil, err
	}
	var meta VersionMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("parse version meta: %w", err)
	}
	return &meta, nil
}

// NormalizeTargetOS maps launcher/device OS names to rule OS names used in Mojang metadata.
func NormalizeTargetOS(os string) string {
	switch strings.ToLower(strings.TrimSpace(os)) {
	case "windows", "win32":
		return "windows"
	case "darwin", "osx", "macos":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

func filterLibraries(libs []Library, targetOS string) []Library {
	out := make([]Library, 0, len(libs))
	for _, lib := range libs {
		if !libraryAllowed(lib, targetOS) {
			continue
		}
		out = append(out, lib)
	}
	return out
}

func libraryAllowed(lib Library, targetOS string) bool {
	return rulesAllow(lib.Rules, targetOS)
}

// defaultRuleFeatures drives Mojang argument/library rule evaluation for our launcher.
var defaultRuleFeatures = map[string]bool{
	"is_demo_user":            false,
	"has_custom_resolution":   true,
	"has_quick_plays_support": false,
}

func ruleMatches(rule Rule, targetOS string) bool {
	if rule.OS != nil && !osMatches(rule.OS.Name, targetOS) {
		return false
	}
	for name, raw := range rule.Features {
		var want bool
		if err := json.Unmarshal(raw, &want); err != nil {
			return false
		}
		have, ok := defaultRuleFeatures[name]
		if !ok {
			have = false
		}
		if want != have {
			return false
		}
	}
	return true
}

func rulesAllow(rules []Rule, targetOS string) bool {
	if len(rules) == 0 {
		return true
	}
	allowed := false
	for _, rule := range rules {
		if !ruleMatches(rule, targetOS) {
			continue
		}
		switch rule.Action {
		case "allow":
			allowed = true
		case "disallow":
			return false
		}
	}
	return allowed
}

func osMatches(name, targetOS string) bool {
	targetOS = NormalizeTargetOS(targetOS)
	switch strings.ToLower(name) {
	case "windows":
		return targetOS == "windows"
	case "osx", "macos":
		return targetOS == "darwin"
	case "linux":
		return targetOS == "linux"
	default:
		return false
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("http %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return io.ReadAll(res.Body)
}
