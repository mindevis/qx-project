package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/testutil"
)

func TestHealthHandler(t *testing.T) {
	db := testutil.OpenSQLiteDB(t)
	h := &HealthHandler{DB: db}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	h.Liveness(c)
	if w.Code != http.StatusOK {
		t.Fatalf("liveness: %d", w.Code)
	}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	h.Readiness(c)
	if w.Code != http.StatusOK {
		t.Fatalf("ready: %d", w.Code)
	}

	testutil.CloseDB(t, db)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	h.Readiness(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable: %d", w.Code)
	}
}
