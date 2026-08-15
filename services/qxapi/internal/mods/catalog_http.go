package mods

import (
	"context"
	"net"
	"net/http"
	"time"
)

const (
	catalogHTTPTimeout  = 8 * time.Second
	catalogPartnerGrace = 1500 * time.Millisecond
	catalogRelaxBudget  = 2 * time.Second
)

func newCatalogHTTPClient() *http.Client {
	return &http.Client{
		Timeout: catalogHTTPTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: catalogHTTPTimeout,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
		},
	}
}

type catalogHalf struct {
	items []SearchItem
	err   error
}

func runCatalogHalf(fn func() ([]SearchItem, error)) <-chan catalogHalf {
	ch := make(chan catalogHalf, 1)
	go func() {
		items, err := fn()
		ch <- catalogHalf{items: items, err: err}
	}()
	return ch
}

func waitPrimaryThenPartner(
	ctx context.Context,
	primary <-chan catalogHalf,
	partner <-chan catalogHalf,
	grace time.Duration,
) (catalogHalf, catalogHalf, bool) {
	var primaryHalf catalogHalf
	select {
	case primaryHalf = <-primary:
	case <-ctx.Done():
		return catalogHalf{err: ctx.Err()}, catalogHalf{}, false
	}

	wait := grace
	if primaryHalf.err != nil {
		wait = catalogHTTPTimeout
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case partnerHalf := <-partner:
		return primaryHalf, partnerHalf, true
	case <-timer.C:
		return primaryHalf, catalogHalf{}, false
	case <-ctx.Done():
		return primaryHalf, catalogHalf{}, false
	}
}
