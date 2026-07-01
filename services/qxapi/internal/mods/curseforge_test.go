package mods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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
