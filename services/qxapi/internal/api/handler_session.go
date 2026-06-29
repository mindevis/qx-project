package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
)

type SessionHandler struct {
	Service *cosmetics.Service
}

type skinServerMeta struct {
	Meta struct {
		ServerName            string `json:"serverName"`
		ImplementationName    string `json:"implementationName"`
		ImplementationVersion string `json:"implementationVersion"`
	} `json:"meta"`
	SkinDomains []string `json:"skinDomains"`
}

func (h *SessionHandler) Meta(c *gin.Context) {
	if h.Service == nil {
		c.Status(http.StatusNotFound)
		return
	}
	domains := h.Service.SkinDomains()
	if len(domains) == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	var resp skinServerMeta
	resp.Meta.ServerName = "QX Skin Server"
	resp.Meta.ImplementationName = "qxapi"
	resp.Meta.ImplementationVersion = "1.0"
	resp.SkinDomains = domains
	c.JSON(http.StatusOK, resp)
}

// Profile serves GET /sessionserver/session/minecraft/profile/{uuid} (Yggdrasil-compatible).
func (h *SessionHandler) Profile(c *gin.Context) {
	rawUUID := strings.TrimSuffix(c.Param("uuid"), ".json")
	body, err := h.Service.SessionProfile(c.Request.Context(), rawUUID, c.Query("name"))
	if err != nil {
		if errors.Is(err, cosmetics.ErrNotFound) {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		if errors.Is(err, cosmetics.ErrValidation) {
			c.Status(http.StatusBadRequest)
			return
		}
		JSONInternal(c)
		return
	}
	c.Data(http.StatusOK, "application/json", body)
}
