package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	BaseURL    string
	DeviceID   string
	HTTPClient *http.Client
}

type RegisterResult struct {
	DeviceID        string    `json:"device_id"`
	Status          string    `json:"status"`
	LinkURL         string    `json:"link_url"`
	PollIntervalSec int       `json:"poll_interval_sec"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type StatusResult struct {
	Status      string  `json:"status"`
	DeviceToken *string `json:"device_token,omitempty"`
	OwnerType   *string `json:"owner_type,omitempty"`
}

func NewClient(baseURL, deviceID string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:3000/api/v1"
	}
	if deviceID == "" {
		deviceID = uuid.NewString()
	}
	httpClient := &http.Client{Timeout: 15 * time.Second}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), DeviceID: deviceID, HTTPClient: httpClient}
}

func (c *Client) Register(ctx context.Context) (*RegisterResult, error) {
	hostname, _ := os.Hostname()
	body, _ := json.Marshal(map[string]string{
		"device_id":        c.DeviceID,
		"os":               runtime.GOOS,
		"hostname":         hostname,
		"launcher_version": "0.1.0",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/launcher/devices/register", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("register failed: %d %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out RegisterResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Status(ctx context.Context) (*StatusResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/launcher/devices/"+c.DeviceID+"/status", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("status failed: %d %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out StatusResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) LinkWithUserToken(ctx context.Context, userToken string) error {
	body, _ := json.Marshal(map[string]string{
		"device_id": c.DeviceID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/launcher/devices/link", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("link failed: %d %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (c *Client) SaveDeviceToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token), 0o600)
}

func (c *Client) ClearDeviceToken(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *Client) Unlink(ctx context.Context, deviceToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/launcher/devices/unlink", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+deviceToken)
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("unlink failed: %d %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func isHTTPUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "401")
}

// UnlinkWithRefresh unlinks the device, refreshing the JWT from /status when needed.
func (c *Client) UnlinkWithRefresh(ctx context.Context, tokenPath string) error {
	token, err := EnsureDeviceToken(ctx, c, tokenPath)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("device not linked")
	}
	if err := c.Unlink(ctx, token); err != nil {
		if !isHTTPUnauthorized(err) {
			return err
		}
		token, refreshErr := RefreshDeviceToken(ctx, c, tokenPath)
		if refreshErr != nil {
			return err
		}
		return c.Unlink(ctx, token)
	}
	return nil
}

func (c *Client) LinkLoop(ctx context.Context, maxPolls int, sleepFn func(time.Duration)) (*StatusResult, error) {
	return c.LinkLoopWithUserToken(ctx, maxPolls, sleepFn, "")
}

func (c *Client) LinkLoopWithUserToken(ctx context.Context, maxPolls int, sleepFn func(time.Duration), userToken string) (*StatusResult, error) {
	if sleepFn == nil {
		sleepFn = time.Sleep
	}
	reg, err := c.Register(ctx)
	if err != nil {
		return nil, err
	}
	return c.PollUntilLinked(ctx, maxPolls, reg.PollIntervalSec, sleepFn, userToken)
}

func (c *Client) PollUntilLinked(ctx context.Context, maxPolls int, pollIntervalSec int, sleepFn func(time.Duration), userToken string) (*StatusResult, error) {
	if sleepFn == nil {
		sleepFn = time.Sleep
	}
	if userToken != "" {
		_ = c.LinkWithUserToken(ctx, userToken)
	}
	interval := time.Duration(pollIntervalSec) * time.Second
	if interval <= 0 {
		interval = 3 * time.Second
	}
	for i := 0; i < maxPolls; i++ {
		status, err := c.Status(ctx)
		if err != nil {
			return nil, err
		}
		if status.Status == "linked" && status.DeviceToken != nil {
			return status, nil
		}
		if i+1 < maxPolls {
			sleepFn(interval)
		}
	}
	return nil, fmt.Errorf("device link timed out after %d polls", maxPolls)
}
