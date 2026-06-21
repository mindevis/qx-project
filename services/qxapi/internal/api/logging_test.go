package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func withLogBuffer(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}

func TestRequestLoggerLevels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		path   string
		status int
		want   string
	}{
		{"health debug", "/api/v1/health", 200, "DEBUG"},
		{"ok info", "/api/v1/users/me", 200, "INFO"},
		{"client warn", "/api/v1/auth/login", 401, "WARN"},
		{"server error", "/api/v1/auth/login", 500, "ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := withLogBuffer(t)
			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(RequestLogger())
			r.GET(tc.path, func(c *gin.Context) { c.Status(tc.status) })

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			out := buf.String()
			if !strings.Contains(out, tc.want) || !strings.Contains(out, "http request") ||
				!strings.Contains(out, `direction=in`) {
				t.Fatalf("log %q, want level %s and direction=in", out, tc.want)
			}
		})
	}
}

func TestRequestLoggerWithQuery(t *testing.T) {
	buf := withLogBuffer(t)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(RequestLogger())
	r.GET("/api/v1/users/me", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me?x=1", nil)
	r.ServeHTTP(w, req)

	if !strings.Contains(buf.String(), "x=1") {
		t.Fatalf("expected query in log: %q", buf.String())
	}
}

func TestRecoveryLogger(t *testing.T) {
	buf := withLogBuffer(t)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(RecoveryLogger())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status: %d", w.Code)
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Fatalf("log: %q", buf.String())
	}
}
