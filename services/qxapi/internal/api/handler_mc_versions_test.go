package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestMcVersionsHandlerList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"latest": {"release":"1.21.4","snapshot":"25w02a"},
			"versions":[
				{"id":"1.21.4","type":"release","url":"https://example/1.21.4.json"},
				{"id":"25w02a","type":"snapshot","url":"https://example/25w02a.json"}
			]
		}`))
	}))
	t.Cleanup(srv.Close)

	h := &McVersionsHandler{
		Client: &mcmanifest.Client{
			ManifestURL: srv.URL,
			HTTPClient:  srv.Client(),
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/launcher/mc-versions", nil)
	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "1.21.4") || !strings.Contains(w.Body.String(), "25w02a") {
		t.Fatalf("body: %s", w.Body.String())
	}
}

func TestMcVersionsHandlerUpstreamError(t *testing.T) {
	h := &McVersionsHandler{
		Client: &mcmanifest.Client{
			ManifestURL: "http://127.0.0.1:1/unreachable",
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/launcher/mc-versions", nil).WithContext(context.Background())
	h.List(c)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status: %d", w.Code)
	}
}
