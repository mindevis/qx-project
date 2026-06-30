package deploy

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const (
	dockerHostGateway     = "host.docker.internal"
	coLocatedAgentAPIBase = "http://127.0.0.1:3000"
)

// remoteAPIResolution is what the dedicated server sees for the public API hostname.
type remoteAPIResolution struct {
	ResolvedIP     string
	RemoteHostname string
}

// agentAPIURL returns the API base URL written into agent.toml on the dedicated server.
// Dev Flow C: SSH to localhost + API on host → agent inside Docker must use host.docker.internal.
// Co-located prod: agent on the same host as the platform stack uses loopback to avoid resolving
// the public hostname via /etc/hosts (Debian maps the hostname to 127.0.1.1, which often breaks WSS).
// Remote dedicated servers always receive QX_PUBLIC_API_URL as-is (never the SSH host name).
func agentAPIURL(apiURL string, cred models.SSHCredential) string {
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = "http://localhost:3000"
	}
	u, err := url.Parse(apiURL)
	if err != nil {
		return apiURL
	}
	if isLocalHost(cred.Host) && isLocalHost(u.Hostname()) {
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
	if isCoLocatedAPI(cred.Host, u.Hostname()) {
		return coLocatedAgentAPIBase
	}
	return apiURL
}

// isCoLocatedAPI is true only when the API hostname resolves to the SSH target IP.
// Hostname string equality is ignored: a remote game server may share the API domain as its
// machine hostname (Debian /etc/hosts → 127.0.1.1) while the API runs on a different host.
func isCoLocatedAPI(sshHost, apiHost string) bool {
	sshHost = strings.ToLower(strings.TrimSpace(sshHost))
	apiHost = strings.ToLower(strings.TrimSpace(apiHost))
	if sshHost == "" || apiHost == "" || isLocalHost(sshHost) {
		return false
	}
	addrs, err := net.LookupHost(apiHost)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if strings.EqualFold(addr, sshHost) {
			return true
		}
	}
	return false
}

func isLocalHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func isLoopbackIP(ip string) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false
	}
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

// apiHostsOverrideScript returns a shell snippet that maps the API hostname to its public IP
// on the dedicated server when local resolution would hit loopback (Debian hostname → 127.0.1.1).
func apiHostsOverrideScript(apiURL string, cred models.SSHCredential, remote remoteAPIResolution) string {
	if !needsAPIHostsOverride(apiURL, cred, remote) {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(apiURL))
	if err != nil {
		return ""
	}
	apiHost := strings.ToLower(strings.TrimSpace(u.Hostname()))
	realIP := firstPublicIP(apiHost)
	if realIP == "" {
		return ""
	}

	// glibc reads /etc/hosts only (not /etc/hosts.d); replace loopback aliases before agent start.
	return fmt.Sprintf(`$SUDO awk -v ip=%q -v h=%q 'BEGIN { skip=0 }
/^# QX agent API$/ { skip=1; next }
skip && /^# end QX agent API$/ { skip=0; next }
skip { next }
{
  for (i=2; i<=NF; i++) if ($i == h) next
  print
}
END {
  print "# QX agent API"
  print ip, h
  print "# end QX agent API"
}' /etc/hosts > /tmp/qx-hosts && $SUDO mv /tmp/qx-hosts /etc/hosts
`, realIP, apiHost)
}

func needsAPIHostsOverride(apiURL string, cred models.SSHCredential, remote remoteAPIResolution) bool {
	u, err := url.Parse(strings.TrimSpace(apiURL))
	if err != nil {
		return false
	}
	apiHost := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if apiHost == "" || isLocalHost(apiHost) {
		return false
	}
	if agentAPIURL(apiURL, cred) == coLocatedAgentAPIBase {
		return false
	}

	remoteIP := strings.TrimSpace(remote.ResolvedIP)
	remoteHostname := strings.ToLower(strings.TrimSpace(remote.RemoteHostname))
	collision := strings.EqualFold(apiHost, cred.Host) ||
		(remoteHostname != "" && remoteHostname == apiHost)
	return collision || isLoopbackIP(remoteIP)
}

func firstPublicIP(host string) string {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if !isLoopbackIP(addr) {
			return addr
		}
	}
	return ""
}
