package cosmetics

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type sessionProfile struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Properties []sessionProperty   `json:"properties"`
}

type sessionProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type texturePayload struct {
	Timestamp   int64                    `json:"timestamp"`
	ProfileID   string                   `json:"profileId"`
	ProfileName string                   `json:"profileName"`
	Textures    map[string]textureEntry  `json:"textures"`
}

type textureEntry struct {
	URL      string            `json:"url"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ProfileTextures describes skin and cape URLs for a game profile UUID.
type ProfileTextures struct {
	Username  string
	GameUUID  string
	SkinModel string
	SkinURL   string
	CapeURL   string
}

// BuildSessionProfile returns a Yggdrasil-compatible session profile JSON body.
// Properties are unsigned (no signature) — works with custom session hosts and authlib-injector.
func BuildSessionProfile(tex ProfileTextures) ([]byte, error) {
	compact := CompactGameUUID(tex.GameUUID)
	if compact == "" {
		return nil, ErrValidation
	}
	name := strings.TrimSpace(tex.Username)
	if name == "" {
		name = "Player"
	}
	textures := map[string]textureEntry{}
	if u := strings.TrimSpace(tex.SkinURL); u != "" {
		entry := textureEntry{URL: u}
		if strings.EqualFold(tex.SkinModel, "alex") {
			entry.Metadata = map[string]string{"model": "slim"}
		}
		textures["SKIN"] = entry
	}
	if u := strings.TrimSpace(tex.CapeURL); u != "" {
		textures["CAPE"] = textureEntry{URL: u}
	}
	if len(textures) == 0 {
		return nil, ErrNotFound
	}
	payload := texturePayload{
		Timestamp:   time.Now().UnixMilli(),
		ProfileID:   compact,
		ProfileName: name,
		Textures:    textures,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	profile := sessionProfile{
		ID:   compact,
		Name: name,
		Properties: []sessionProperty{{
			Name:  "textures",
			Value: base64.StdEncoding.EncodeToString(raw),
		}},
	}
	return json.Marshal(profile)
}
