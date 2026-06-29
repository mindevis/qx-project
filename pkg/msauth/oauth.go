package msauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AuthorizeRequest struct {
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type AuthorizeURL struct {
	URL      string
	State    string
	Verifier string
}

type MSATokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (c *Client) BeginAuthorize() (*AuthorizeURL, error) {
	verifier, challenge, err := NewPKCE()
	if err != nil {
		return nil, err
	}
	state := mustRandomState()
	u, err := c.authorizeURL(AuthorizeRequest{
		State:               state,
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		return nil, err
	}
	return &AuthorizeURL{URL: u, State: state, Verifier: verifier}, nil
}

func (c *Client) authorizeURL(req AuthorizeRequest) (string, error) {
	if c.cfg.RedirectURI == "" {
		return "", fmt.Errorf("missing redirect uri")
	}
	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", c.cfg.RedirectURI)
	q.Set("scope", oauthScope)
	q.Set("state", req.State)
	q.Set("code_challenge", req.CodeChallenge)
	q.Set("code_challenge_method", req.CodeChallengeMethod)
	q.Set("prompt", "select_account")
	return msAuthorizeURL + "?" + q.Encode(), nil
}

func (c *Client) ExchangeCode(ctx context.Context, code, verifier string) (*MSATokens, error) {
	form := url.Values{}
	form.Set("client_id", c.cfg.ClientID)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.cfg.RedirectURI)
	form.Set("code_verifier", verifier)
	if secret := strings.TrimSpace(c.cfg.ClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}
	return c.postToken(ctx, form)
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (*MSATokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("missing refresh token")
	}
	form := url.Values{}
	form.Set("client_id", c.cfg.ClientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	if secret := strings.TrimSpace(c.cfg.ClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}
	return c.postToken(ctx, form)
}

func (c *Client) postToken(ctx context.Context, form url.Values) (*MSATokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, msTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ms token: http %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("ms token: missing access_token")
	}
	if parsed.RefreshToken == "" {
		parsed.RefreshToken = form.Get("refresh_token")
	}
	return &MSATokens{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresIn:    parsed.ExpiresIn,
	}, nil
}

type Profile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capes    []any  `json:"capes,omitempty"`
	Skins    []any  `json:"skins,omitempty"`
	LinkedAt time.Time
}

type Session struct {
	AccessToken string
	UUID        string
	Username    string
}

func (c *Client) Login(ctx context.Context, msaAccessToken string) (*Session, error) {
	xbl, err := c.authenticateXBL(ctx, msaAccessToken)
	if err != nil {
		return nil, err
	}
	xsts, err := c.authorizeXSTS(ctx, xbl.Token)
	if err != nil {
		return nil, err
	}
	mcToken, err := c.loginMinecraft(ctx, xsts)
	if err != nil {
		return nil, err
	}
	profile, err := c.FetchProfile(ctx, mcToken)
	if err != nil {
		return nil, err
	}
	return &Session{
		AccessToken: mcToken,
		UUID:        profile.ID,
		Username:    profile.Name,
	}, nil
}

func (c *Client) LoginFromRefresh(ctx context.Context, refreshToken string) (*Session, *MSATokens, error) {
	tokens, err := c.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, nil, err
	}
	session, err := c.Login(ctx, tokens.AccessToken)
	if err != nil {
		return nil, nil, err
	}
	return session, tokens, nil
}

func (c *Client) FetchProfile(ctx context.Context, mcAccessToken string) (*Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mcProfileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+mcAccessToken)
	res, err := c.HTTP().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("minecraft profile not found — account may not own the game")
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("minecraft profile: http %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}
	if profile.ID == "" || profile.Name == "" {
		return nil, fmt.Errorf("minecraft profile incomplete")
	}
	return &profile, nil
}

type xblToken struct {
	Token string
	UHS   string
}

func (c *Client) authenticateXBL(ctx context.Context, msaAccessToken string) (*xblToken, error) {
	payload := map[string]any{
		"Properties": map[string]string{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + msaAccessToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}
	var resp struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			Xui []struct {
				UHS string `json:"uhs"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
	}
	if err := c.postJSON(ctx, xblAuthURL, payload, &resp); err != nil {
		return nil, fmt.Errorf("xbl auth: %w", err)
	}
	if resp.Token == "" || len(resp.DisplayClaims.Xui) == 0 {
		return nil, fmt.Errorf("xbl auth: incomplete response")
	}
	return &xblToken{Token: resp.Token, UHS: resp.DisplayClaims.Xui[0].UHS}, nil
}

type xstsToken struct {
	Token string
	UHS   string
}

func (c *Client) authorizeXSTS(ctx context.Context, xblToken string) (*xstsToken, error) {
	payload := map[string]any{
		"Properties": map[string]any{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xblToken},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}
	var resp struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			Xui []struct {
				UHS string `json:"uhs"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
	}
	if err := c.postJSON(ctx, xstsAuthURL, payload, &resp); err != nil {
		return nil, fmt.Errorf("xsts auth: %w", err)
	}
	if resp.Token == "" || len(resp.DisplayClaims.Xui) == 0 {
		return nil, fmt.Errorf("xsts auth: incomplete response")
	}
	return &xstsToken{Token: resp.Token, UHS: resp.DisplayClaims.Xui[0].UHS}, nil
}

func (c *Client) loginMinecraft(ctx context.Context, xsts *xstsToken) (string, error) {
	payload := map[string]string{
		"identityToken": "XBL3.0 x=" + xsts.UHS + ";" + xsts.Token,
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.postJSON(ctx, mcLoginURL, payload, &resp); err != nil {
		return "", fmt.Errorf("minecraft login: %w", err)
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("minecraft login: missing access_token")
	}
	return resp.AccessToken, nil
}

func (c *Client) postJSON(ctx context.Context, endpoint string, payload any, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.HTTP().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// GameUUID returns the dashed UUID in the format expected by modern Minecraft manifests.
func GameUUID(id string) string {
	id = strings.ReplaceAll(strings.TrimSpace(id), "-", "")
	if len(id) != 32 {
		return id
	}
	return id[0:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:32]
}

// GameUUIDCompact returns UUID without dashes for legacy launch arguments.
func GameUUIDCompact(id string) string {
	return strings.ReplaceAll(GameUUID(id), "-", "")
}
