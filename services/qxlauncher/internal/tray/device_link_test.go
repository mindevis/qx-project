package tray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/services/qxlauncher/internal/device"
)

type stubMenu struct{ disabled bool }

func (m *stubMenu) disable() { m.disabled = true }

func TestCompleteDeviceLink(t *testing.T) {
	linked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/launcher/devices/link":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
			linked = true
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/launcher/devices/dev-1/status":
			if !linked {
				_ = json.NewEncoder(w).Encode(device.StatusResult{Status: "pending_link"})
				return
			}
			token := "device-token"
			_ = json.NewEncoder(w).Encode(device.StatusResult{Status: "linked", DeviceToken: &token})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := device.NewClient(srv.URL+"/api/v1", "dev-1")
	tokenPath := filepath.Join(t.TempDir(), "device_token")
	reg := &device.RegisterResult{
		DeviceID:        "dev-1",
		UserCode:        "ABCD-1234",
		PollIntervalSec: 1,
	}

	token, err := CompleteDeviceLink(context.Background(), client, DeviceLinkConfig{
		TokenPath:    tokenPath,
		MaxLinkPolls: 3,
		UserToken:    "user-jwt",
	}, reg)
	if err != nil || token != "device-token" {
		t.Fatalf("complete link: err=%v token=%q", err, token)
	}
	b, err := os.ReadFile(tokenPath)
	if err != nil || string(b) != "device-token" {
		t.Fatalf("saved token: %q err=%v", b, err)
	}
}

func TestPollDeviceLinkUsesPendingRegister(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/launcher/devices/register":
			t.Fatal("unexpected register when pending register provided")
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/launcher/devices/dev-9/status":
			token := "tok-9"
			_ = json.NewEncoder(w).Encode(device.StatusResult{Status: "linked", DeviceToken: &token})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := device.NewClient(srv.URL+"/api/v1", "dev-9")
	tokenPath := filepath.Join(t.TempDir(), "device_token")
	reg := &device.RegisterResult{
		DeviceID:        "dev-9",
		LinkURL:         "http://localhost/link",
		UserCode:        "WXYZ-9999",
		PollIntervalSec: 1,
	}
	linkURL := ""
	menu := &stubMenu{}
	loopStarted := ""
	pollDeviceLink(context.Background(), SystrayConfig{
		DeviceClient:    client,
		TokenPath:       tokenPath,
		MaxLinkPolls:    3,
		PendingRegister: reg,
	}, menu, &linkURL, func(token string) {
		loopStarted = token
	}, nil)

	if !menu.disabled {
		t.Fatal("expected link menu disabled")
	}
	if loopStarted != "tok-9" {
		t.Fatalf("start loop token: %q", loopStarted)
	}
	if linkURL != reg.LinkURL {
		t.Fatalf("link url: %q", linkURL)
	}
}
