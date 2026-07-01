package mojang

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrSessionRevoked     = errors.New("mojang refresh token revoked or expired")
	ErrSessionUnavailable = errors.New("mojang auth service unavailable")
)

// ClassifyAuthError maps Microsoft/Mojang OAuth and login failures to stable API errors.
// Transient upstream/network failures must not be reported as a revoked session.
func ClassifyAuthError(err error) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "invalid_grant"),
		strings.Contains(msg, "invalid_refresh_token"),
		strings.Contains(msg, "refresh token has expired"),
		strings.Contains(msg, "interaction_required"):
		return fmt.Errorf("%w: %v", ErrSessionRevoked, err)
	case strings.Contains(msg, "invalid_client"),
		strings.Contains(msg, "unauthorized_client"):
		return fmt.Errorf("%w: %v", ErrNotConfigured, err)
	case strings.Contains(msg, "ms token: http 401"),
		strings.Contains(msg, "ms token: http 403"):
		return fmt.Errorf("%w: %v", ErrSessionRevoked, err)
	case strings.Contains(msg, "ms token: http 400"):
		return fmt.Errorf("%w: %v", ErrSessionRevoked, err)
	case strings.Contains(msg, "http 5"):
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	default:
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
}
