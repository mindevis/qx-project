package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type Client struct {
	BaseURL     string
	DeviceToken string
	HTTPClient  *http.Client
}

func New(baseURL, deviceToken string) *Client {
	return &Client{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		DeviceToken: deviceToken,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

type MojangSession struct {
	Username    string `json:"username"`
	UUID        string `json:"uuid"`
	AccessToken string `json:"access_token"`
}

type LaunchCosmetics struct {
	SkinModel      string `json:"skin_model"`
	SkinURL        string `json:"skin_url,omitempty"`
	UseSkinServer  bool   `json:"use_skin_server,omitempty"`
	SkinServerHost string `json:"skin_server_host,omitempty"`
	GameUUID       string `json:"game_uuid,omitempty"`
}

type LaunchRequestItem struct {
	ID         string                           `json:"id"`
	Status     string                           `json:"status"`
	InstanceID string                           `json:"instance_id"`
	Manifest   *mcmanifest.InstanceLaunchManifest `json:"manifest"`
	Profile    *OfflineProfile                  `json:"profile"`
	Mojang     *MojangSession                   `json:"mojang_session"`
	Cosmetics  *LaunchCosmetics                 `json:"cosmetics"`
}

type OfflineProfile struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	OfflineUUID string `json:"offline_uuid"`
	Model       string `json:"model"`
}

type InstanceItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MCVersion string `json:"mc_version"`
	Loader    string `json:"loader"`
}

type LaunchRequestCreated struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	InstanceID string `json:"instance_id"`
}

type UpdateRequestItem struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
}

func (c *Client) FetchPendingLaunch(ctx context.Context) (*LaunchRequestItem, error) {
	body, err := c.request(ctx, http.MethodGet, "/launcher/launch-requests/pending", nil, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Item *LaunchRequestItem `json:"item"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Item, nil
}

func (c *Client) UpdateLaunch(ctx context.Context, id string, payload map[string]any) error {
	b, _ := json.Marshal(payload)
	_, err := c.request(ctx, http.MethodPatch, "/launcher/launch-requests/"+id, b, true)
	return err
}

func (c *Client) FetchPendingUpdate(ctx context.Context) (*UpdateRequestItem, error) {
	body, err := c.request(ctx, http.MethodGet, "/launcher/update-requests/pending", nil, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Item *UpdateRequestItem `json:"item"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Item, nil
}

func (c *Client) CompleteUpdate(ctx context.Context, id, status, launcherVersion, errorCode string) error {
	payload := map[string]any{"status": status}
	if launcherVersion != "" {
		payload["launcher_version"] = launcherVersion
	}
	if errorCode != "" {
		payload["error_code"] = errorCode
	}
	b, _ := json.Marshal(payload)
	_, err := c.request(ctx, http.MethodPatch, "/launcher/update-requests/"+id, b, true)
	return err
}

func (c *Client) SetDeviceToken(token string) {
	c.DeviceToken = strings.TrimSpace(token)
}

func (c *Client) PingDevice(ctx context.Context) error {
	_, err := c.request(ctx, http.MethodGet, "/launcher/devices/me", nil, true)
	return err
}

func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), ": 401 ")
}

// IsUnavailable reports whether err is a transient network or connectivity failure.
func IsUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, sub := range []string{
		"connection refused",
		"connectex",
		"actively refused",
		"no such host",
		"network is unreachable",
		"connection reset",
		"i/o timeout",
	} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}

func (c *Client) FetchDeviceInstances(ctx context.Context) ([]InstanceItem, error) {
	body, err := c.request(ctx, http.MethodGet, "/launcher/devices/me/instances", nil, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Items []InstanceItem `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) ListProfiles(ctx context.Context, userToken string) ([]OfflineProfile, error) {
	body, err := c.requestWithToken(ctx, http.MethodGet, "/launcher/profiles", nil, userToken)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Items []OfflineProfile `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) CreateLaunchRequest(ctx context.Context, userToken, instanceID, offlineProfileID string, useMojangAccount bool) (*LaunchRequestCreated, error) {
	payload := map[string]any{
		"instance_id": instanceID,
	}
	if offlineProfileID != "" {
		payload["offline_profile_id"] = offlineProfileID
	}
	if useMojangAccount {
		payload["use_mojang_account"] = true
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	body, err := c.requestWithToken(ctx, http.MethodPost, "/launcher/launch-requests", b, userToken)
	if err != nil {
		return nil, err
	}
	var created LaunchRequestCreated
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) DeleteInstance(ctx context.Context, userToken, instanceID string) error {
	_, err := c.requestWithToken(ctx, http.MethodDelete, "/instances/"+instanceID, nil, userToken)
	return err
}

func (c *Client) request(ctx context.Context, method, path string, body []byte, deviceAuth bool) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if deviceAuth && c.DeviceToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.DeviceToken)
	}
	return c.do(req)
}

func (c *Client) requestWithToken(ctx context.Context, method, path string, body []byte, token string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("api %s %s: %d %s", req.Method, req.URL.Path, res.StatusCode, strings.TrimSpace(string(b)))
	}
	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	return b, nil
}
