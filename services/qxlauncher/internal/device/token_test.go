package device

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDeviceTokenRefreshesFromStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/launcher/devices/me":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"UNAUTHORIZED"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/launcher/devices/dev-1/status":
			token := "fresh-device-jwt"
			_ = json.NewEncoder(w).Encode(StatusResult{Status: "linked", DeviceToken: &token})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "device_token")
	if err := os.WriteFile(tokenPath, []byte("stale-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveDeviceID(dir, "dev-1"); err != nil {
		t.Fatal(err)
	}

	client := NewClient(srv.URL+"/api/v1", "dev-1")
	token, err := EnsureDeviceToken(context.Background(), client, tokenPath)
	if err != nil || token != "fresh-device-jwt" {
		t.Fatalf("ensure: err=%v token=%q", err, token)
	}
	saved, err := os.ReadFile(tokenPath)
	if err != nil || string(saved) != "fresh-device-jwt" {
		t.Fatalf("saved token: %q err=%v", saved, err)
	}
}

func TestEnsureDeviceTokenKeepsValidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/launcher/devices/me" {
			_, _ = w.Write([]byte(`{"device_id":"dev-1","status":"linked"}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "device_token")
	if err := os.WriteFile(tokenPath, []byte("valid-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := NewClient(srv.URL+"/api/v1", "dev-1")
	token, err := EnsureDeviceToken(context.Background(), client, tokenPath)
	if err != nil || token != "valid-token" {
		t.Fatalf("ensure: err=%v token=%q", err, token)
	}
}
