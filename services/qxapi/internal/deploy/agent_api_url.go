package deploy

import (
	"net"
	"net/url"
	"strings"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const dockerHostGateway = "host.docker.internal"

// agentAPIURL returns the API base URL written into agent.toml on the VPS.
// Dev Flow C: SSH to localhost + API on host → agent inside Docker must use host.docker.internal.
func agentAPIURL(apiURL string, cred models.SSHCredential) string {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = "http://localhost:3000"
	}
	u, err := url.Parse(apiURL)
	if err != nil {
		return apiURL
	}
	if !isLocalHost(cred.Host) || !isLocalHost(u.Hostname()) {
		return apiURL
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" || u.Scheme == "wss" {
			port = "443"
		} else {
			port = "3000"
		}
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	u.Host = net.JoinHostPort(dockerHostGateway, port)
	return u.String()
}

func isLocalHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}
