package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPendingLaunch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/launch-requests/pending" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"item": map[string]any{
				"id":          "req-1",
				"status":      "dispatched",
				"instance_id": "inst-1",
				"manifest": map[string]any{
					"instance_id": "inst-1",
					"mc_version":  "1.21",
					"main_class":  "net.minecraft.client.main.Main",
					"client_jar":  map[string]any{"url": "https://example/jar", "sha1": "abc"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, "device-token")
	item, err := client.FetchPendingLaunch(context.Background())
	if err != nil || item == nil || item.Manifest == nil {
		t.Fatalf("pending: err=%v item=%+v", err, item)
	}
}

func TestFetchDeviceInstances(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/launcher/devices/me/instances" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "1", "name": "A", "mc_version": "1.21", "loader": "vanilla"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, "device-token")
	items, err := client.FetchDeviceInstances(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("instances: err=%v items=%v", err, items)
	}
}
