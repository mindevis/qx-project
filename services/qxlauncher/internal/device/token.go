package device

import (
	"context"
	"fmt"
	"strings"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
)

// EnsureDeviceToken returns a valid device JWT, refreshing from the status endpoint when needed.
func EnsureDeviceToken(ctx context.Context, client *Client, tokenPath string) (string, error) {
	token := ReadToken(tokenPath)
	if token != "" {
		api := apiclient.New(client.BaseURL, token)
		if err := api.PingDevice(ctx); err == nil {
			return token, nil
		}
	}

	status, err := client.Status(ctx)
	if err != nil {
		return "", err
	}
	if status.Status != "linked" || status.DeviceToken == nil {
		if token != "" {
			_ = client.ClearDeviceToken(tokenPath)
		}
		return "", nil
	}

	token = strings.TrimSpace(*status.DeviceToken)
	if token == "" {
		return "", nil
	}
	if err := client.SaveDeviceToken(tokenPath, token); err != nil {
		return "", err
	}
	return token, nil
}

// RefreshDeviceToken fetches a fresh device JWT for a linked device.
func RefreshDeviceToken(ctx context.Context, client *Client, tokenPath string) (string, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return "", err
	}
	if status.Status != "linked" || status.DeviceToken == nil {
		return "", fmt.Errorf("device not linked")
	}
	token := strings.TrimSpace(*status.DeviceToken)
	if token == "" {
		return "", fmt.Errorf("empty device token")
	}
	if err := client.SaveDeviceToken(tokenPath, token); err != nil {
		return "", err
	}
	return token, nil
}
