package mods

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNewServiceCatalogClientTimeout(t *testing.T) {
	t.Parallel()
	svc := NewService(Config{ModrinthUserAgent: "QXTest/1.0", CurseForgeAPIKey: "k"})
	if svc.modrinth.httpClient.Timeout != catalogHTTPTimeout {
		t.Fatalf("modrinth timeout: got %s want %s", svc.modrinth.httpClient.Timeout, catalogHTTPTimeout)
	}
	if svc.curseforge.httpClient.Timeout != catalogHTTPTimeout {
		t.Fatalf("curseforge timeout: got %s want %s", svc.curseforge.httpClient.Timeout, catalogHTTPTimeout)
	}
}

func TestBrowseBothReturnsModrinthWhenCurseForgeHangs(t *testing.T) {
	t.Parallel()

	mrSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hits":[{"project_id":"sodium","slug":"sodium","title":"Sodium","description":"fps","author":"dev","icon_url":"","project_type":"mod","downloads":100,"versions":["1.21.1"],"categories":["neoforge"]}]}`))
	}))
	t.Cleanup(mrSrv.Close)

	cfSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(cfSrv.Close)

	mrBase, err := url.Parse(mrSrv.URL)
	if err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		modrinth: &modrinthClient{
			httpClient: &http.Client{
				Timeout:   catalogHTTPTimeout,
				Transport: rewriteTransport{base: mrBase, rt: http.DefaultTransport},
			},
			userAgent: "QXTest/1.0",
		},
		curseforge: &curseForgeClient{
			httpClient: &http.Client{Timeout: catalogHTTPTimeout},
			apiKey:     "test-key",
			apiBase:    cfSrv.URL + "/v1",
		},
	}

	started := time.Now()
	items, _, err := svc.Browse(context.Background(), ProjectTypeMod, "neoforge", "1.21.1", "all", "downloads", 20, 0)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if len(items) != 1 || items[0].ID != "sodium" {
		t.Fatalf("expected modrinth item, got %+v", items)
	}
	if elapsed > 6*time.Second {
		t.Fatalf("browse waited too long for hanging curseforge: %s", elapsed)
	}
}

func TestWaitPrimaryThenPartnerUsesGrace(t *testing.T) {
	t.Parallel()
	primary := runCatalogHalf(func() ([]SearchItem, error) {
		return []SearchItem{{ID: "mr"}}, nil
	})
	partner := runCatalogHalf(func() ([]SearchItem, error) {
		time.Sleep(50 * time.Millisecond)
		return []SearchItem{{ID: "cf"}}, nil
	})
	mr, cf, ok := waitPrimaryThenPartner(context.Background(), primary, partner, 200*time.Millisecond)
	if !ok || mr.items[0].ID != "mr" || cf.items[0].ID != "cf" {
		t.Fatalf("expected both halves, ok=%v mr=%+v cf=%+v", ok, mr, cf)
	}
}

func TestCatalogResultCacheable(t *testing.T) {
	t.Parallel()
	items := []SearchItem{{ID: "x"}}
	if catalogResultCacheable(true, false, nil, items, nil) {
		t.Fatal("timed-out curseforge partner must not be cached as a complete page")
	}
	if !catalogResultCacheable(true, true, nil, items, nil) {
		t.Fatal("complete mixed page should be cached")
	}
	if catalogResultCacheable(true, true, nil, nil, nil) {
		t.Fatal("empty page should not be cached")
	}
}
