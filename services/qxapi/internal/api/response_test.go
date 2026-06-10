package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestJSONHelpers(t *testing.T) {
	tests := []struct {
		name   string
		call   func(*gin.Context)
		status int
		code   string
	}{
		{"validation", func(c *gin.Context) { JSONValidation(c, "bad") }, http.StatusBadRequest, "VALIDATION_ERROR"},
		{"unauthorized", func(c *gin.Context) { JSONUnauthorized(c) }, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"conflict", func(c *gin.Context) { JSONConflict(c, "dup") }, http.StatusConflict, "CONFLICT"},
		{"internal", func(c *gin.Context) { JSONInternal(c) }, http.StatusInternalServerError, "INTERNAL"},
		{"custom", func(c *gin.Context) { JSONError(c, 418, "TEAPOT", "brew") }, 418, "TEAPOT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.call(c)
			if w.Code != tt.status {
				t.Fatalf("status: got %d", w.Code)
			}
			var body ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("json: %v", err)
			}
			if body.Error.Code != tt.code {
				t.Fatalf("code: got %s", body.Error.Code)
			}
		})
	}
}
