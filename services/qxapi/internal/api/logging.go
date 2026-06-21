package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	qxlog "github.com/qxproject/qx/pkg/log"
)

func requestPath(c *gin.Context) string {
	path := c.Request.URL.Path
	if q := c.Request.URL.RawQuery; q != "" {
		return path + "?" + q
	}
	return path
}

func logHTTPRequest(status int, path string, attrs []any) {
	if strings.HasPrefix(path, "/api/v1/health") {
		slog.Debug("http request", attrs...)
		return
	}
	switch {
	case status >= http.StatusInternalServerError:
		slog.Error("http request", attrs...)
	case status >= http.StatusBadRequest:
		slog.Warn("http request", attrs...)
	default:
		slog.Info("http request", attrs...)
	}
}

// RequestLogger logs HTTP requests with level by status code.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := requestPath(c)
		c.Next()
		logHTTPRequest(c.Writer.Status(), path, []any{
			"direction", qxlog.DirectionIn,
			"transport", qxlog.TransportHTTP,
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
			"client_ip", c.ClientIP(),
		})
	}
}

// RecoveryLogger logs panics and returns 500.
func RecoveryLogger() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("panic recovered",
			"error", recovered,
			"method", c.Request.Method,
			"path", requestPath(c),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
