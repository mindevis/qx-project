package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
)

type ReleaseHandler struct {
	Service *launcher.Service
}

func (h *ReleaseHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, h.Service.GetRelease())
}
