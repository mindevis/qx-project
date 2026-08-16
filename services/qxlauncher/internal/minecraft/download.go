package minecraft

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/qxproject/qx/pkg/safepath"
)

const (
	downloadMaxAttempts = 5
	downloadRetryBase   = 2 * time.Second
)

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 2 * time.Minute,
		},
	}
}

func (d *Downloader) httpClient() *http.Client {
	if d.HTTPClient != nil {
		return d.HTTPClient
	}
	return defaultHTTPClient()
}

func isRetryableDownloadError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "tls handshake timeout"),
		strings.Contains(msg, "connection reset"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "eof"):
		return true
	default:
		return false
	}
}

func isRetryableDownloadStatus(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

func (d *Downloader) downloadOnce(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := d.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: status %d", url, res.StatusCode)
	}
	dest, err = safepath.ResolveRoot(dest)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(dest); err != nil {
		return err
	}
	return safepath.WriteStreamAtomic(dest, res.Body)
}

func (d *Downloader) downloadWithRetry(ctx context.Context, url, dest string) error {
	var lastErr error
	for attempt := 1; attempt <= downloadMaxAttempts; attempt++ {
		err := d.downloadOnce(ctx, url, dest)
		if err == nil {
			return nil
		}
		lastErr = err
		retry := false
		if isRetryableDownloadError(err) {
			retry = true
		} else {
			for _, code := range []int{
				http.StatusTooManyRequests,
				http.StatusBadGateway,
				http.StatusServiceUnavailable,
				http.StatusGatewayTimeout,
			} {
				if strings.Contains(err.Error(), fmt.Sprintf("status %d", code)) && isRetryableDownloadStatus(code) {
					retry = true
					break
				}
			}
		}
		if !retry || attempt == downloadMaxAttempts {
			return err
		}
		wait := downloadRetryBase * time.Duration(1<<(attempt-1))
		d.progressf("download", "retry %d/%d for %s in %s (%v)", attempt, downloadMaxAttempts-1, filepath.Base(dest), wait, err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return lastErr
}
