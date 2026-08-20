package mods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSpigetSearchFindsProtocolLib(t *testing.T) {
	t.Parallel()
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/search/resources/") {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("spiget requires a user-agent")
		}
		got, _ = url.ParseQuery(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1997,"name":"ProtocolLib","tag":"packets","downloads":3413519,"premium":false,"external":true,"existenceStatus":1,"testedVersions":["1.21"],"file":{"type":"external","externalUrl":"https://github.com/dmulloy2/ProtocolLib/releases/download/5.4.0/ProtocolLib.jar"},"icon":{"url":""},"author":{"id":618}},
			{"id":2,"name":"Premium","premium":true,"external":false,"existenceStatus":1,"file":{"type":".jar"}},
			{"id":3,"name":"Skript","file":{"type":".sk"},"premium":false,"existenceStatus":1}
		]`))
	}))
	t.Cleanup(srv.Close)

	c := &spigetClient{httpClient: srv.Client(), userAgent: "qx-test", apiBase: srv.URL}
	items, err := c.search(context.Background(), "ProtocolLib", ProjectTypePlugin, "paper", "26.2", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "ProtocolLib" || items[0].Source != SourceSpigot {
		t.Fatalf("items: %+v", items)
	}
	if items[0].ExternalURL != "https://www.spigotmc.org/resources/1997/" {
		t.Fatalf("url: %s", items[0].ExternalURL)
	}
	if got.Get("size") == "" {
		t.Fatal("expected size")
	}
}

func TestSpigetSearchIgnoresNonPlugins(t *testing.T) {
	t.Parallel()
	c := &spigetClient{httpClient: http.DefaultClient, userAgent: "qx-test"}
	items, err := c.search(context.Background(), "sodium", ProjectTypeMod, "fabric", "1.21", 20)
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("expected no spiget mods, got %+v", items)
	}
}

func TestSpigetVersionUsesHostedDownload(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/resources/28140":
			_, _ = w.Write([]byte(`{"id":28140,"name":"LuckPerms","premium":false,"external":false,"existenceStatus":1,"testedVersions":["1.21"],"file":{"type":".jar"}}`))
		case strings.HasSuffix(r.URL.Path, "/versions"):
			_, _ = w.Write([]byte(`[{"id":123,"name":"5.4.0","releaseDate":1710000000}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &spigetClient{httpClient: srv.Client(), userAgent: "qx-test", apiBase: srv.URL}
	versions, err := c.listVersions(context.Background(), "28140", "paper", "1.21")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("versions: %+v", versions)
	}
	if versions[0].Files[0].URL != "https://api.spiget.org/v2/resources/28140/versions/123/download" {
		t.Fatalf("download: %s", versions[0].Files[0].URL)
	}
}

func TestSpigetVersionPrefersExternalJar(t *testing.T) {
	t.Parallel()
	resource := spigetResource{
		ID:       1997,
		Name:     "ProtocolLib",
		External: true,
		File:     spigetFile{ExternalURL: "https://github.com/dmulloy2/ProtocolLib/releases/download/5.4.0/ProtocolLib.jar"},
	}
	ver := spigetVersionFrom(resource, spigetVersion{ID: 602511, Name: "5.4.0"})
	if ver == nil {
		t.Fatal("expected version")
	}
	if ver.Files[0].URL != resource.File.ExternalURL {
		t.Fatalf("url: %s", ver.Files[0].URL)
	}
	if ver.Files[0].Filename != "ProtocolLib.jar" {
		t.Fatalf("filename: %s", ver.Files[0].Filename)
	}
}

func TestLooksLikePluginDownload(t *testing.T) {
	t.Parallel()
	if !looksLikePluginDownload("https://github.com/dmulloy2/ProtocolLib/releases/download/5.4.0/ProtocolLib.jar") {
		t.Fatal("github jar")
	}
	if looksLikePluginDownload("https://exceptionflug.de/protocolize") {
		t.Fatal("webpage should not count as a plugin file")
	}
}
