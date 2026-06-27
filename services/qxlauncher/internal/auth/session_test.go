package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginAndSessionStorage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(Session{
			AccessToken:  "access",
			RefreshToken: "refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	t.Cleanup(srv.Close)

	s, err := Login(context.Background(), srv.URL, "u@test.com", "password123")
	if err != nil || s.AccessToken != "access" {
		t.Fatalf("login: err=%v s=%+v", err, s)
	}
	if s.SavedAt == 0 {
		t.Fatal("expected saved_at on login")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := SaveSession(path, s); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadSession(path)
	if err != nil || loaded.AccessToken != "access" {
		t.Fatalf("load: err=%v loaded=%+v", err, loaded)
	}
}

func TestRefreshAndEnsureFresh(t *testing.T) {
	var refreshCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/refresh":
			refreshCalls++
			_ = json.NewEncoder(w).Encode(Session{
				AccessToken:  "access-new",
				RefreshToken: "refresh-new",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	refreshed, err := Refresh(context.Background(), srv.URL, "refresh-old")
	if err != nil || refreshed.AccessToken != "access-new" {
		t.Fatalf("refresh: err=%v s=%+v", err, refreshed)
	}
	refreshCalls = 0

	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	stale := &Session{
		AccessToken:  "access-old",
		RefreshToken: "refresh-old",
		ExpiresIn:    60,
		SavedAt:      time.Now().UTC().Add(-2 * time.Hour).Unix(),
	}
	if err := SaveSession(path, stale); err != nil {
		t.Fatal(err)
	}
	token, err := EnsureFreshAccessToken(context.Background(), srv.URL, path)
	if err != nil || token != "access-new" || refreshCalls != 1 {
		t.Fatalf("ensure fresh: token=%q err=%v refreshCalls=%d", token, err, refreshCalls)
	}

	fresh := &Session{
		AccessToken:  "still-valid",
		RefreshToken: "refresh-old",
		ExpiresIn:    3600,
		SavedAt:      time.Now().UTC().Unix(),
	}
	if err := SaveSession(path, fresh); err != nil {
		t.Fatal(err)
	}
	token, err = EnsureFreshAccessToken(context.Background(), srv.URL, path)
	if err != nil || token != "still-valid" || refreshCalls != 1 {
		t.Fatalf("skip refresh: token=%q err=%v refreshCalls=%d", token, err, refreshCalls)
	}
}

func TestEnsureFreshAccessTokenOfflineUsesStaleToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	stale := &Session{
		AccessToken:  "offline-access",
		RefreshToken: "refresh-old",
		ExpiresIn:    60,
		SavedAt:      time.Now().UTC().Add(-2 * time.Hour).Unix(),
	}
	if err := SaveSession(path, stale); err != nil {
		t.Fatal(err)
	}
	token, err := EnsureFreshAccessToken(context.Background(), "http://127.0.0.1:1", path)
	if err != nil || token != "offline-access" {
		t.Fatalf("offline ensure: token=%q err=%v", token, err)
	}
}

func TestLoadSessionMissingFile(t *testing.T) {
	if _, err := LoadSession(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadSessionInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSession(path); err == nil {
		t.Fatal("expected json error")
	}
}
