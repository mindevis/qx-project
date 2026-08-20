package mods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHangarSearchFindsProtocolLib(t *testing.T) {
	t.Parallel()
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("hangar requires a user-agent")
		}
		got, _ = url.ParseQuery(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[{"id":433,"name":"ProtocolLib","description":"packets","avatarUrl":"https://hangarcdn.papermc.io/avatars/project/433.webp","namespace":{"owner":"dmulloy2","slug":"ProtocolLib"},"stats":{"downloads":25163},"supportedPlatforms":{"PAPER":["1.21.8"]}}]}`))
	}))
	t.Cleanup(srv.Close)

	c := &hangarClient{httpClient: srv.Client(), userAgent: "qx-test", apiBase: srv.URL}
	items, err := c.search(context.Background(), "ProtocolLib", ProjectTypePlugin, "paper", "26.2", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "ProtocolLib" || items[0].Source != SourceHangar {
		t.Fatalf("items: %+v", items)
	}
	if items[0].ExternalURL != "https://hangar.papermc.io/dmulloy2/ProtocolLib" {
		t.Fatalf("url: %s", items[0].ExternalURL)
	}
	if got.Get("q") != "ProtocolLib" {
		t.Fatalf("query: %s", got.Get("q"))
	}
	if got.Get("version") != "" {
		t.Fatalf("named search must not filter hangar by minecraft version, got %q", got.Get("version"))
	}
	if got.Get("platform") != "PAPER" {
		t.Fatalf("platform: %s", got.Get("platform"))
	}
}

func TestHangarSearchIgnoresNonPlugins(t *testing.T) {
	t.Parallel()
	c := &hangarClient{httpClient: http.DefaultClient, userAgent: "qx-test"}
	items, err := c.search(context.Background(), "sodium", ProjectTypeMod, "fabric", "1.21", 20)
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("expected no hangar mods, got %+v", items)
	}
}

func TestHangarVersionPrefersExternalGitHubDownload(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/versions") {
			_, _ = w.Write([]byte(`{"result":[{"id":18184,"name":"5.4.0","createdAt":"2025-08-07T05:05:45Z","downloads":{"PAPER":{"fileInfo":null,"externalUrl":"https://github.com/dmulloy2/ProtocolLib/releases/download/5.4.0/ProtocolLib.jar","downloadUrl":null}},"platformDependencies":{"PAPER":["1.21.8"]}}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	c := &hangarClient{httpClient: srv.Client(), userAgent: "qx-test", apiBase: srv.URL}
	versions, err := c.listVersions(context.Background(), "dmulloy2/ProtocolLib", "paper", "26.2")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("versions: %+v", versions)
	}
	if versions[0].Files[0].URL != "https://github.com/dmulloy2/ProtocolLib/releases/download/5.4.0/ProtocolLib.jar" {
		t.Fatalf("download: %s", versions[0].Files[0].URL)
	}
	if versions[0].Files[0].Filename != "ProtocolLib.jar" {
		t.Fatalf("filename: %s", versions[0].Files[0].Filename)
	}
}

func TestHangarProjectSlug(t *testing.T) {
	t.Parallel()
	if got := hangarProjectSlug("dmulloy2/ProtocolLib"); got != "ProtocolLib" {
		t.Fatalf("got %q", got)
	}
	if got := hangarProjectSlug("ProtocolLib"); got != "ProtocolLib" {
		t.Fatalf("got %q", got)
	}
}
