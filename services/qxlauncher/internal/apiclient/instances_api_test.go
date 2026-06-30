package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateLaunchRequest(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/launcher/launch-requests" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer user-jwt" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "req-1",
			"status":      "queued",
			"instance_id": "inst-1",
		})
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, "device-token")
	created, err := client.CreateLaunchRequest(context.Background(), "user-jwt", "inst-1", "prof-1", false)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "req-1" || gotBody["instance_id"] != "inst-1" || gotBody["offline_profile_id"] != "prof-1" {
		t.Fatalf("unexpected create: %+v body=%v", created, gotBody)
	}
}

func TestDeleteInstance(t *testing.T) {
	var deleted string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/instances/inst-9" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer user-jwt" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		deleted = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, "device-token")
	if err := client.DeleteInstance(context.Background(), "user-jwt", "inst-9"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted != "/instances/inst-9" {
		t.Fatalf("deleted path: %q", deleted)
	}
}

func TestListProfiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/launcher/profiles" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "p1", "username": "Steve", "offline_uuid": "00000000-0000-0000-0000-000000000001"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := New(srv.URL, "device-token")
	profiles, err := client.ListProfiles(context.Background(), "user-jwt")
	if err != nil || len(profiles) != 1 || profiles[0].Username != "Steve" {
		t.Fatalf("profiles: err=%v items=%+v", err, profiles)
	}
}
