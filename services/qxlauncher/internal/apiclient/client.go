package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type LaunchRequestItem struct {
	ID         string                           `json:"id"`
	Status     string                           `json:"status"`
	InstanceID string                           `json:"instance_id"`
	Manifest   *mcmanifest.InstanceLaunchManifest `json:"manifest"`
	Profile    *OfflineProfile                  `json:"profile"`
}

type OfflineProfile struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	OfflineUUID string `json:"offline_uuid"`
}

type InstanceItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MCVersion string `json:"mc_version"`
	Loader    string `json:"loader"`
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

func (c *Client) ListInstances(ctx context.Context, userToken string) ([]InstanceItem, error) {
	body, err := c.requestWithToken(ctx, http.MethodGet, "/instances", nil, userToken)
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
