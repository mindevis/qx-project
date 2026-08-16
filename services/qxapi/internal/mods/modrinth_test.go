package mods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type rewriteTransport struct {
	base *url.URL
	rt   http.RoundTripper
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	newReq.URL.Scheme = t.base.Scheme
	newReq.URL.Host = t.base.Host
	newReq.Host = t.base.Host
	return t.rt.RoundTrip(newReq)
}

func TestModrinthSearchIconURL(t *testing.T) {
	t.Parallel()
	if got := modrinthSearchIconURL("https://cdn/icon.png", ""); got != "https://cdn/icon.png" {
		t.Fatalf("icon_url: got %q", got)
	}
	if got := modrinthSearchIconURL("", "https://cdn/legacy.png"); got != "https://cdn/legacy.png" {
		t.Fatalf("display_icon_url fallback: got %q", got)
	}
	if got := modrinthSearchIconURL("https://cdn/new.png", "https://cdn/old.png"); got != "https://cdn/new.png" {
		t.Fatalf("prefer icon_url: got %q", got)
	}
}

func TestModrinthListVersionsFallsBackWithoutMCVersion(t *testing.T) {
	t.Parallel()

	calls := make([]url.Values, 0, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q, _ := url.ParseQuery(r.URL.RawQuery)
		calls = append(calls, q)
		w.Header().Set("Content-Type", "application/json")
		switch len(calls) {
		case 1, 2:
			_, _ = w.Write([]byte(`[]`))
		default:
			_, _ = w.Write([]byte(`[{"id":"ver-1","version_number":"1.0.0","game_versions":["1.21.1"],"loaders":["fabric"],"date_published":"2024-01-01","dependencies":[],"files":[{"filename":"mod.jar","url":"https://example.com/mod.jar","size":123,"hashes":{"sha1":"abc"}}]}]`))
		}
	}))
	defer srv.Close()

	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &modrinthClient{
		httpClient: &http.Client{Transport: rewriteTransport{base: base, rt: http.DefaultTransport}},
		userAgent:  "qx-test",
	}

	versions, err := client.listVersions(context.Background(), "project-123", "fabric", "1.21")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected fallback versions, got %d", len(versions))
	}
	if versions[0].ID != "ver-1" {
		t.Fatalf("unexpected version id: %q", versions[0].ID)
	}
	if len(calls) != 3 {
		t.Fatalf("expected 3 API calls, got %d", len(calls))
	}
	if !strings.Contains(calls[0].Get("loaders"), "fabric") {
		t.Fatalf("expected first call to include loader, got %q", calls[0].Get("loaders"))
	}
	if calls[1].Get("loaders") != "" {
		t.Fatalf("expected second call to drop loader filter, got %q", calls[1].Get("loaders"))
	}
	if calls[2].Get("loaders") != "" || calls[2].Get("game_versions") != "" {
		t.Fatalf("expected third call to request all versions, got loaders=%q game_versions=%q", calls[2].Get("loaders"), calls[2].Get("game_versions"))
	}
}

func TestNormalizeDependencyType(t *testing.T) {
	t.Parallel()
	if got := normalizeDependencyType("incompatible"); got != "incompatible" {
		t.Fatalf("incompatible: got %q", got)
	}
	if got := normalizeDependencyType("optional"); got != "optional" {
		t.Fatalf("optional: got %q", got)
	}
	if !skipCatalogDependency("incompatible") || !skipCatalogDependency("embedded") {
		t.Fatal("should skip incompatible and embedded catalog dependencies")
	}
	if skipCatalogDependency("required") || skipCatalogDependency("optional") {
		t.Fatal("should keep required and optional catalog dependencies")
	}
}
