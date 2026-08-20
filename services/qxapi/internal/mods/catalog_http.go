package mods

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	catalogHTTPTimeout  = 8 * time.Second
	catalogPartnerGrace = 5 * time.Second
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

func waitOptionalHalf(ctx context.Context, ch <-chan catalogHalf, grace time.Duration) (catalogHalf, bool) {
	if ch == nil {
		return catalogHalf{}, false
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case half := <-ch:
		return half, true
	case <-timer.C:
		return catalogHalf{}, false
	case <-ctx.Done():
		return catalogHalf{}, false
	}
}

func collectOptionalHalves(ctx context.Context, chans []<-chan catalogHalf, grace time.Duration) []catalogHalf {
	n := 0
	for _, ch := range chans {
		if ch != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	merged := make(chan catalogHalf, n)
	var wg sync.WaitGroup
	for _, ch := range chans {
		if ch == nil {
			continue
		}
		wg.Add(1)
		go func(ch <-chan catalogHalf) {
			defer wg.Done()
			select {
			case half := <-ch:
				select {
				case merged <- half:
				case <-ctx.Done():
				}
			case <-ctx.Done():
			}
		}(ch)
	}
	go func() {
		wg.Wait()
		close(merged)
	}()

	timer := time.NewTimer(grace)
	defer timer.Stop()
	out := make([]catalogHalf, 0, n)
	for {
		select {
		case half, ok := <-merged:
			if !ok {
				return out
			}
			out = append(out, half)
			if len(out) == n {
				return out
			}
		case <-timer.C:
			return out
		case <-ctx.Done():
			return out
		}
	}
}

func mergeOptionalCatalogs(items []SearchItem, extras []catalogHalf, limit int) []SearchItem {
	for _, extra := range extras {
		if extra.err != nil || len(extra.items) == 0 {
			continue
		}
		if len(items) == 0 {
			items = extra.items[:min(len(extra.items), limit)]
			continue
		}
		items = pairAndInterleaveSearch(items, extra.items, limit)
	}
	return items
}
