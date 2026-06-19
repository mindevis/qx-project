package mcmanifest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveVersionAndBuildManifest(t *testing.T) {
	versionJSON := []byte(`{
		"id": "1.21",
		"type": "release",
		"mainClass": "net.minecraft.client.main.Main",
		"assets": "1.21",
		"assetIndex": {"id":"1.21","sha1":"abc","size":1,"totalSize":2,"url":"https://example/assets/1.21.json"},
		"downloads": {"client": {"sha1":"jar","size":100,"url":"https://example/client.jar"}},
		"libraries": [{"name":"com:test:1","downloads":{"artifact":{"sha1":"lib","size":1,"url":"https://example/lib.jar"}}}],
		"javaVersion": {"component":"java-runtime-delta","majorVersion":21}
	}`)

	var versionURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_, _ = w.Write([]byte(`{"latest":{"release":"1.21"},"versions":[{"id":"1.21","type":"release","url":"` + versionURL + `"}]}`))
		case "/1.21.json":
			_, _ = w.Write(versionJSON)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	versionURL = srv.URL + "/1.21.json"

	client := &Client{
		ManifestURL: srv.URL + "/manifest.json",
		HTTPClient:  srv.Client(),
	}

	url, err := client.ResolveVersionURL(context.Background(), "1.21")
	if err != nil || url == "" {
		t.Fatalf("resolve: %v %q", err, url)
	}

	meta, err := client.FetchVersionMeta(context.Background(), url)
	if err != nil || meta.MainClass == "" {
		t.Fatalf("meta: %v %+v", err, meta)
	}

	launch, err := client.BuildInstanceManifest(context.Background(), "inst-1", "Test", "1.21", "vanilla")
	if err != nil || launch.ClientJar.URL == "" {
		t.Fatalf("launch manifest: %v %+v", err, launch)
	}
	if launch.JavaComponent != "java-runtime-delta" || launch.JavaMajor != 21 {
		t.Fatalf("java fields: %+v", launch)
	}
}

func TestFilterLibrariesDisallow(t *testing.T) {
	old := currentOSName
	currentOSName = func() string { return "windows" }
	t.Cleanup(func() { currentOSName = old })

	libs := filterLibraries([]Library{
		{Name: "allowed", Downloads: &LibraryDownloads{Artifact: &DownloadFile{URL: "u"}}},
		{
			Name:      "linux-only",
			Rules:     []Rule{{Action: "allow", OS: &RuleOS{Name: "linux"}}},
			Downloads: &LibraryDownloads{Artifact: &DownloadFile{URL: "u"}},
		},
	})
	if len(libs) != 1 || libs[0].Name != "allowed" {
		t.Fatalf("filter: %+v", libs)
	}
}

func TestResolveVersionNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"versions":[]}`))
	}))
	t.Cleanup(srv.Close)
	client := &Client{ManifestURL: srv.URL, HTTPClient: srv.Client()}
	if _, err := client.ResolveVersionURL(context.Background(), "9.99"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestResolveVersionEmptyID(t *testing.T) {
	client := NewClient()
	if _, err := client.ResolveVersionURL(context.Background(), ""); err == nil {
		t.Fatal("expected validation error")
	}
}
