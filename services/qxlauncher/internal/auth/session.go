package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
)

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	SavedAt      int64  `json:"saved_at,omitempty"`
}

func (s *Session) expiresAt() time.Time {
	if s.SavedAt <= 0 {
		return time.Time{}
	}
	return time.Unix(s.SavedAt, 0).Add(time.Duration(s.ExpiresIn) * time.Second)
}

func (s *Session) needsRefresh() bool {
	if s.SavedAt <= 0 || s.ExpiresIn <= 0 {
		return false
	}
	return time.Now().UTC().Add(30 * time.Second).After(s.expiresAt())
}

func LoadSession(path string) (*Session, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.AccessToken == "" {
		return nil, fmt.Errorf("empty access token")
	}
	return &s, nil
}

func SaveSession(path string, s *Session) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if s.SavedAt == 0 {
		s.SavedAt = time.Now().UTC().Unix()
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func Login(ctx context.Context, baseURL, email, password string) (*Session, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/auth/login", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed: %d %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.AccessToken == "" {
		return nil, fmt.Errorf("login response missing access_token")
	}
	s.SavedAt = time.Now().UTC().Unix()
	return &s, nil
}

func Refresh(ctx context.Context, baseURL, refreshToken string) (*Session, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %d %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	if s.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}
	s.SavedAt = time.Now().UTC().Unix()
	return &s, nil
}

func EnsureFreshAccessToken(ctx context.Context, baseURL, sessionPath string) (string, error) {
	session, err := LoadSession(sessionPath)
	if err != nil {
		return "", err
	}
	if !session.needsRefresh() {
		return session.AccessToken, nil
	}
	if session.RefreshToken == "" {
		return session.AccessToken, nil
	}
	refreshed, err := Refresh(ctx, baseURL, session.RefreshToken)
	if err != nil {
		if apiclient.IsUnavailable(err) {
			return session.AccessToken, nil
		}
		return session.AccessToken, err
	}
	if err := SaveSession(sessionPath, refreshed); err != nil {
		return refreshed.AccessToken, err
	}
	return refreshed.AccessToken, nil
}
