package cosmetics

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFetchMojangSkin(t *testing.T) {
	t.Parallel()
	skinPNG := mustTestSkinPNG(t, 64, 64)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/profiles/minecraft/Steve":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"8667ba71b85a4004af5ef5b72f0b7bad"}`))
		case r.URL.Path == "/session/minecraft/profile/8667ba71b85a4004af5ef5b72f0b7bad":
			skinFileURL := "http://" + r.Host + "/skin.png"
			payload, _ := json.Marshal(map[string]any{
				"textures": map[string]any{
					"SKIN": map[string]any{
						"url": skinFileURL,
					},
				},
			})
			encoded := base64.StdEncoding.EncodeToString(payload)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"properties":[{"name":"textures","value":"` + encoded + `"}]}`))
		case r.URL.Path == "/skin.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(skinPNG)
		case r.URL.Path == "/users/profiles/minecraft/Missing":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	origProfile := mojangProfileURL
	origSession := mojangSessionURL
	origExtra := extraSkinDownloadHosts
	mojangProfileURL = srv.URL + "/users/profiles/minecraft/"
	mojangSessionURL = srv.URL + "/session/minecraft/profile/"
	extraSkinDownloadHosts = []string{httptestHost(srv.URL)}
	t.Cleanup(func() {
		mojangProfileURL = origProfile
		mojangSessionURL = origSession
		extraSkinDownloadHosts = origExtra
	})

	got, err := FetchMojangSkin(context.Background(), client, "Steve")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(got.PNG) == 0 {
		t.Fatal("expected png bytes")
	}
	if got.SkinModel != "steve" {
		t.Fatalf("model: %q", got.SkinModel)
	}

	if _, err := FetchMojangSkin(context.Background(), client, "Missing"); err != ErrPlayerNotFound {
		t.Fatalf("missing player: %v", err)
	}
}

func httptestHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func TestSanitizeMojangLookupURL(t *testing.T) {
	t.Parallel()
	if _, err := sanitizeMojangLookupURL(mojangProfileURL, "../evil", minecraftUsernameRe); err == nil {
		t.Fatal("expected invalid username")
	}
	got, err := sanitizeMojangLookupURL(mojangProfileURL, "Steve", minecraftUsernameRe)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://api.mojang.com/users/profiles/minecraft/Steve" {
		t.Fatalf("url: %s", got)
	}
}

func TestSanitizeSkinDownloadURL(t *testing.T) {
	t.Parallel()
	if _, err := sanitizeSkinDownloadURL("http://127.0.0.1/skin.png"); err == nil {
		t.Fatal("expected localhost rejected")
	}
	got, err := sanitizeSkinDownloadURL("http://textures.minecraft.net/texture/abc")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://textures.minecraft.net/texture/abc" {
		t.Fatalf("url: %s", got)
	}
}

func mustTestSkinPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 120, B: 80, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
