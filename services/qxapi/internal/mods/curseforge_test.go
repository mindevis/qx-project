package mods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCurseForgeBrowseQueryParams(t *testing.T) {
	t.Parallel()
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/mods/search" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		got, _ = url.ParseQuery(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":42,"name":"JEI","slug":"jei","summary":"items","downloadCount":1,"logo":{"thumbnailUrl":""},"authors":[{"name":"meili"}],"latestFilesIndexes":[{"gameVersion":"1.20.1","modLoader":1}]}]}`))
	}))
	t.Cleanup(srv.Close)

	c := &curseForgeClient{
		httpClient: srv.Client(),
		apiKey:     "test-key",
		apiBase:    srv.URL + "/v1",
	}
	items, err := c.browse(context.Background(), ProjectTypeMod, "forge", "1.20.1", "downloads", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items len %d", len(items))
	}
	if got.Get("modLoaderType") != "1" {
		t.Fatalf("modLoaderType: got %q want 1", got.Get("modLoaderType"))
	}
	if got.Get("gameVersion") != "1.20.1" {
		t.Fatalf("gameVersion: got %q", got.Get("gameVersion"))
	}
	if got.Get("sortOrder") != "desc" {
		t.Fatalf("sortOrder: got %q want desc", got.Get("sortOrder"))
	}
	if got.Get("sortField") != "6" {
		t.Fatalf("sortField: got %q want 6", got.Get("sortField"))
	}
	if items[0].Loaders[0] != "forge" {
		t.Fatalf("loader name: got %q", items[0].Loaders[0])
	}
}

func TestCurseForgeBrowseEmptyQuery(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ := url.ParseQuery(r.URL.RawQuery)
		if got.Get("searchFilter") != "" {
			t.Fatalf("searchFilter should be empty for browse, got %q", got.Get("searchFilter"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"A","slug":"a","summary":"","downloadCount":0,"logo":{"thumbnailUrl":""},"authors":[],"latestFilesIndexes":[]}]}`))
	}))
	t.Cleanup(srv.Close)

	c := &curseForgeClient{
		httpClient: srv.Client(),
		apiKey:     "test-key",
		apiBase:    srv.URL + "/v1",
	}
	items, err := c.browse(context.Background(), ProjectTypeMod, "forge", "1.20.1", "downloads", 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected browse with empty searchFilter to return items, got %d", len(items))
	}
}

func TestCurseForgeDisabledReturnsNil(t *testing.T) {
	t.Parallel()
	c := &curseForgeClient{apiKey: ""}
	items, err := c.browse(context.Background(), ProjectTypeMod, "forge", "1.20.1", "downloads", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("expected nil items when disabled, got %v", items)
	}
}

func TestServiceBrowseCurseForgeRequiresKey(t *testing.T) {
	t.Parallel()
	svc := NewService(Config{})
	_, _, err := svc.Browse(context.Background(), ProjectTypeMod, "forge", "1.20.1", SourceCurseForge, "downloads", 10, 0)
	if err == nil {
		t.Fatal("expected error when curseforge key missing and source=curseforge")
	}
}

func TestCurseForgeLoaderMappings(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"forge":    "1",
		"fabric":   "4",
		"quilt":    "5",
		"neoforge": "6",
	}
	for loader, want := range cases {
		if got := curseForgeLoaderType(loader); got != want {
			t.Fatalf("%s: got %q want %q", loader, got, want)
		}
	}
	if curseForgeLoaderName(4) != "fabric" {
		t.Fatalf("fabric name mapping")
	}
}

func TestCurseForgeClassIDDatapack(t *testing.T) {
	t.Parallel()
	if got := curseForgeClassID(ProjectTypeDatapack); got != 6945 {
		t.Fatalf("datapack classId: got %d want 6945", got)
	}
}

func TestCurseForgeSidesFromGameVersions(t *testing.T) {
	t.Parallel()
	client, server := curseForgeSidesFromGameVersions([]string{"Client", "1.20.1", "Forge", "Server"})
	if client != "required" || server != "required" {
		t.Fatalf("both sides: client=%q server=%q", client, server)
	}
	client, server = curseForgeSidesFromGameVersions([]string{"Client", "1.20.1"})
	if client != "required" || server != "unsupported" {
		t.Fatalf("client only: client=%q server=%q", client, server)
	}
	client, server = curseForgeSidesFromGameVersions([]string{"Server", "1.20.1"})
	if client != "unsupported" || server != "required" {
		t.Fatalf("server only: client=%q server=%q", client, server)
	}
}

func TestCurseForgeSearchMapsClientServerSides(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":42,"name":"JEI","slug":"jei","summary":"items","downloadCount":1,"logo":{"thumbnailUrl":""},"authors":[{"name":"meili"}],"latestFilesIndexes":[{"gameVersion":"1.20.1","modLoader":1}],"latestFiles":[{"gameVersions":["1.20.1","Forge"],"modLoader":1},{"gameVersions":["Client","Server","1.20.1","Forge"],"modLoader":1}]}]}`))
	}))
	t.Cleanup(srv.Close)

	c := &curseForgeClient{
		httpClient: srv.Client(),
		apiKey:     "test-key",
		apiBase:    srv.URL + "/v1",
	}
	items, err := c.browse(context.Background(), ProjectTypeMod, "forge", "1.20.1", "downloads", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items len %d", len(items))
	}
	if items[0].ClientSide != "required" || items[0].ServerSide != "required" {
		t.Fatalf("sides: client=%q server=%q", items[0].ClientSide, items[0].ServerSide)
	}
}

func TestCurseForgeSearchMapsServerPackSide(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":7,"name":"Pack","slug":"pack","summary":"","downloadCount":0,"logo":{"thumbnailUrl":""},"authors":[],"latestFilesIndexes":[],"latestFiles":[{"gameVersions":["1.20.1","Forge"],"modLoader":1,"isServerPack":true}]}]}`))
	}))
	t.Cleanup(srv.Close)

	c := &curseForgeClient{
		httpClient: srv.Client(),
		apiKey:     "test-key",
		apiBase:    srv.URL + "/v1",
	}
	items, err := c.browse(context.Background(), ProjectTypeMod, "forge", "1.20.1", "downloads", 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].ClientSide != "unsupported" || items[0].ServerSide != "required" {
		t.Fatalf("server pack sides: client=%q server=%q", items[0].ClientSide, items[0].ServerSide)
	}
}

func TestCurseForgeGetVersionUsesInlineDependencies(t *testing.T) {
	t.Parallel()
	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/mods/32274/files/5789363":
			_, _ = w.Write([]byte(`{"data":{"id":5789363,"displayName":"5.10.3","fileName":"journeymap-1.20.1-5.10.3-forge.jar","fileDate":"2024-01-01","gameVersions":["1.20.1"],"modLoader":1,"downloadUrl":"https://example/mod.jar","fileLength":123,"hashes":[{"value":"abc","algo":1}],"dependencies":[{"modId":999,"relationType":3}]}}`))
		case r.URL.Path == "/v1/mods/999":
			_, _ = w.Write([]byte(`{"data":{"id":999,"name":"Dep Mod","slug":"dep","summary":"","description":"","downloadCount":0,"logo":{"thumbnailUrl":""},"authors":[]}}`))
		case r.URL.Path == "/v1/mods/999/files":
			_, _ = w.Write([]byte(`{"data":[{"id":111,"displayName":"1.0","fileName":"dep.jar","fileDate":"2024-01-01","gameVersions":["1.20.1"],"modLoader":1,"downloadUrl":"https://example/dep.jar","fileLength":10,"hashes":[]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &curseForgeClient{
		httpClient: srv.Client(),
		apiKey:     "test-key",
		apiBase:    srv.URL + "/v1",
	}
	version, err := c.getVersion(context.Background(), "32274", "5789363", "forge", "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if version.Files[0].Filename != "journeymap-1.20.1-5.10.3-forge.jar" {
		t.Fatalf("filename: got %q", version.Files[0].Filename)
	}
	if len(version.Dependencies) != 1 || version.Dependencies[0].ProjectID != "999" {
		t.Fatalf("dependencies: %+v", version.Dependencies)
	}
	for _, path := range gotPaths {
		if strings.HasSuffix(path, "/dependencies") {
			t.Fatalf("should not call removed dependencies endpoint, got %q", path)
		}
	}
}

func TestCurseForgeGetVersionFallsBackToList(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/mods/32274/files/5789363":
			http.NotFound(w, r)
		case "/v1/mods/32274/files":
			_, _ = w.Write([]byte(`{"data":[{"id":5789363,"displayName":"5.10.3","fileName":"journeymap-1.20.1-5.10.3-forge.jar","fileDate":"2024-01-01","gameVersions":["1.20.1"],"modLoader":1,"downloadUrl":"https://example/mod.jar","fileLength":123,"hashes":[]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &curseForgeClient{
		httpClient: srv.Client(),
		apiKey:     "test-key",
		apiBase:    srv.URL + "/v1",
	}
	version, err := c.getVersion(context.Background(), "32274", "5789363", "forge", "1.20.1")
	if err != nil {
		t.Fatal(err)
	}
	if version.ID != "5789363" {
		t.Fatalf("version id: got %q", version.ID)
	}
}

func TestCurseForgeHTTPError(t *testing.T) {
	t.Parallel()
	err := &CurseForgeHTTPError{StatusCode: http.StatusNotFound, Body: "missing"}
	if !isCurseForgeHTTPStatus(err, http.StatusNotFound) {
		t.Fatal("expected 404 match")
	}
	if isCurseForgeHTTPStatus(err, http.StatusForbidden) {
		t.Fatal("expected no 403 match")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("error string: %q", err.Error())
	}
}
