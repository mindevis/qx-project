package msauth

import (
	"net/http"
	"time"
)

// DefaultMinecraftClientID is the public Xbox/Minecraft launcher application id.
const DefaultMinecraftClientID = "00000000402b5328"

const (
	msAuthorizeURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize"
	msTokenURL     = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	xblAuthURL     = "https://user.auth.xboxlive.com/user/authenticate"
	xstsAuthURL    = "https://xsts.auth.xboxlive.com/xsts/authorize"
	mcLoginURL     = "https://api.minecraftservices.com/authentication/login_with_xbox"
	mcProfileURL   = "https://api.minecraftservices.com/minecraft/profile"
	oauthScope     = "XboxLive.signin offline_access"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.ClientID == "" {
		cfg.ClientID = DefaultMinecraftClientID
	}
	hc := &http.Client{Timeout: 30 * time.Second}
	return &Client{cfg: cfg, httpClient: hc}
}

func (c *Client) HTTP() *http.Client {
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return c.httpClient
}
