package device

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRegisterStatusAndLinkLoop(t *testing.T) {
	linked := false
	linkCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/launcher/devices/register":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(RegisterResult{
				DeviceID:        "dev-1",
				Status:          "pending_link",
				UserCode:        "ABCD-1234",
				LinkURL:         "http://localhost/launcher/link?device=dev-1",
				PollIntervalSec: 1,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/launcher/devices/link":
			linkCalled = r.Header.Get("Authorization") == "Bearer user-jwt"
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
			linked = true
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/launcher/devices/dev-1/status":
			if !linked {
				_ = json.NewEncoder(w).Encode(StatusResult{Status: "pending_link"})
				return
			}
			token := "device-token"
			_ = json.NewEncoder(w).Encode(StatusResult{Status: "linked", DeviceToken: &token})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL+"/api/v1", "dev-1")
	ctx := context.Background()

	got, err := client.LinkLoopWithUserToken(ctx, 2, func(time.Duration) {}, "user-jwt")
	if err != nil || got.DeviceToken == nil || *got.DeviceToken != "device-token" {
		t.Fatalf("link loop: err=%v got=%+v", err, got)
	}
	if !linkCalled {
		t.Fatal("expected user link call")
	}
}

func TestLinkWithUserToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "dev-2")
	if err := client.LinkWithUserToken(context.Background(), "tok", "ABCD-1234"); err != nil {
		t.Fatalf("link: %v", err)
	}
}

func TestRegisterStatusAndLinkLoopGuest(t *testing.T) {
	linked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/launcher/devices/register":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(RegisterResult{
				DeviceID:        "dev-1",
				Status:          "pending_link",
				UserCode:        "ABCD-1234",
				LinkURL:         "http://localhost/launcher/link?device=dev-1",
				PollIntervalSec: 1,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/launcher/devices/dev-1/status":
			if !linked {
				_ = json.NewEncoder(w).Encode(StatusResult{Status: "pending_link"})
				return
			}
			token := "device-token"
			_ = json.NewEncoder(w).Encode(StatusResult{Status: "linked", DeviceToken: &token})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL+"/api/v1", "dev-1")
	ctx := context.Background()

	reg, err := client.Register(ctx)
	if err != nil || reg.UserCode == "" {
		t.Fatalf("register: %v %+v", err, reg)
	}

	status, err := client.Status(ctx)
	if err != nil || status.Status != "pending_link" {
		t.Fatalf("status pending: %v %+v", err, status)
	}

	linked = true
	got, err := client.LinkLoop(ctx, 2, func(time.Duration) {})
	if err != nil || got.DeviceToken == nil || *got.DeviceToken != "device-token" {
		t.Fatalf("link loop: %v %+v", err, got)
	}
}

func TestSaveDeviceToken(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/token"
	client := NewClient("", "dev")
	if err := client.SaveDeviceToken(path, "secret"); err != nil {
		t.Fatalf("save: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "secret" {
		t.Fatalf("read back: %v %q", err, b)
	}
	if err := client.ClearDeviceToken(path); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected removed: %v", err)
	}
	if err := client.ClearDeviceToken(path); err != nil {
		t.Fatalf("clear missing: %v", err)
	}
}

func TestUnlink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/launcher/devices/unlink" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer device-tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "pending_link"})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL+"/api/v1", "dev")
	if err := client.Unlink(context.Background(), "device-tok"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
}
