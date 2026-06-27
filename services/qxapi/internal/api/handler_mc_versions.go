package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

type McVersionsHandler struct {
	Client *mcmanifest.Client
}

func (h *McVersionsHandler) List(c *gin.Context) {
	client := h.Client
	if client == nil {
		client = mcmanifest.NewClient()
	}
	result, err := client.ListVersions(c.Request.Context())
	if err != nil {
		JSONError(c, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "failed to fetch Minecraft versions")
		return
	}
	c.JSON(http.StatusOK, result)
}
