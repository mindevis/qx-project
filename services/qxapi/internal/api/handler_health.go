package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/database"
)

type HealthHandler struct {
	DB *gorm.DB
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	if err := database.Ping(h.DB); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "db": "down"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
