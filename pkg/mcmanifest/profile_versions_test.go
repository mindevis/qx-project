package mcmanifest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveLatestFabricLoaderVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"loader":{"version":"0.16.10","stable":true}},
			{"loader":{"version":"0.16.14","stable":true}},
			{"loader":{"version":"0.16.12","stable":true}}
		]`))
	}))
	t.Cleanup(srv.Close)

	client := &Client{HTTPClient: srv.Client()}
	got, err := client.resolveLatestProfileLoaderVersion(context.Background(), "1.20.1", srv.URL+"/%s", "fabric")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "0.16.14" {
		t.Fatalf("got %q want 0.16.14", got)
	}
}

func TestResolveLatestQuiltLoaderVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"loader":{"version":"0.29.3-beta.1","stable":true}},
			{"loader":{"version":"0.28.1","stable":true}},
			{"loader":{"version":"0.28.0","stable":true}}
		]`))
	}))
	t.Cleanup(srv.Close)

	client := &Client{HTTPClient: srv.Client()}
	got, err := client.resolveLatestProfileLoaderVersion(context.Background(), "1.21.1", srv.URL+"/%s", "quilt")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "0.28.1" {
		t.Fatalf("got %q want 0.28.1", got)
	}
}
