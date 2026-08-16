package cosmetics

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var ErrPlayerNotFound = errors.New("minecraft player not found")

var (
	mojangProfileURL = "https://api.mojang.com/users/profiles/minecraft/"
	mojangSessionURL = "https://sessionserver.mojang.com/session/minecraft/profile/"
)

var (
	minecraftUsernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)
	minecraftUUIDRe     = regexp.MustCompile(`^[a-fA-F0-9-]{32,36}$`)
)

// extraSkinDownloadHosts is for tests (httptest). Production only allows textures.minecraft.net.
var extraSkinDownloadHosts []string

type MojangSkinResult struct {
	PNG       []byte
	SkinModel string
}

type mojangProfileLookup struct {
	ID string `json:"id"`
}

type mojangSessionProfile struct {
	Properties []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"properties"`
}

type mojangTexturePayload struct {
	Textures struct {
		Skin struct {
			URL      string `json:"url"`
			Metadata struct {
				Model string `json:"model"`
			} `json:"metadata"`
		} `json:"SKIN"`
	} `json:"textures"`
}

// FetchMojangSkin downloads the equipped skin for a Minecraft username.
func FetchMojangSkin(ctx context.Context, client *http.Client, username string) (*MojangSkinResult, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrValidation
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	uuid, err := mojangUUID(ctx, client, username)
	if err != nil {
		return nil, err
	}
	skinURL, model, err := mojangSkinURL(ctx, client, uuid)
	if err != nil {
		return nil, err
	}
	png, err := downloadSkinPNG(ctx, client, skinURL)
	if err != nil {
		return nil, err
	}
	if err := ValidateSkinPNG(png); err != nil {
		return nil, err
	}
	return &MojangSkinResult{PNG: png, SkinModel: model}, nil
}

func sanitizeMojangLookupURL(base, id string, idRe *regexp.Regexp) (string, error) {
	if !idRe.MatchString(id) {
		return "", ErrValidation
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" {
		return "", ErrValidation
	}
	return parsed.JoinPath(id).String(), nil
}

func sanitizeSkinDownloadURL(raw string) (string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Host == "" {
		return "", ErrValidation
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "textures.minecraft.net" {
		safe := &url.URL{Scheme: "https", Host: "textures.minecraft.net", Path: parsed.EscapedPath()}
		return safe.String(), nil
	}
	for _, extra := range extraSkinDownloadHosts {
		if host == extra {
			return parsed.String(), nil
		}
	}
	return "", ErrValidation
}

func mojangUUID(ctx context.Context, client *http.Client, username string) (string, error) {
	lookupURL, err := sanitizeMojangLookupURL(mojangProfileURL, username, minecraftUsernameRe)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lookupURL, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusNoContent {
		return "", ErrPlayerNotFound
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mojang profile lookup: %s", res.Status)
	}
	var profile mojangProfileLookup
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&profile); err != nil {
		return "", err
	}
	if profile.ID == "" {
		return "", ErrPlayerNotFound
	}
	return profile.ID, nil
}

func mojangSkinURL(ctx context.Context, client *http.Client, uuid string) (string, string, error) {
	lookupURL, err := sanitizeMojangLookupURL(mojangSessionURL, uuid, minecraftUUIDRe)
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lookupURL, nil)
	if err != nil {
		return "", "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return "", "", ErrPlayerNotFound
	}
	if res.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("mojang session profile: %s", res.Status)
	}
	var profile mojangSessionProfile
	if err := json.NewDecoder(io.LimitReader(res.Body, 32*1024)).Decode(&profile); err != nil {
		return "", "", err
	}
	for _, prop := range profile.Properties {
		if prop.Name != "textures" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(prop.Value)
		if err != nil {
			return "", "", err
		}
		var payload mojangTexturePayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return "", "", err
		}
		url := strings.TrimSpace(payload.Textures.Skin.URL)
		if url == "" {
			return "", "", ErrPlayerNotFound
		}
		model := models.CosmeticsSkinModelSteve
		if strings.EqualFold(payload.Textures.Skin.Metadata.Model, "slim") {
			model = models.CosmeticsSkinModelAlex
		}
		return url, model, nil
	}
	return "", "", ErrPlayerNotFound
}

func downloadSkinPNG(ctx context.Context, client *http.Client, skinURL string) ([]byte, error) {
	skinURL, err := sanitizeSkinDownloadURL(skinURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, skinURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skin download: %s", res.Status)
	}
	return io.ReadAll(io.LimitReader(res.Body, maxSkinBytes))
}
