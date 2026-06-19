package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/services/qxapi/internal/agenthub"
	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

var agentUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type AgentWSHandler struct {
	Hub     *agenthub.Hub
	Tokens  *auth.TokenService
	Servers *servers.Service
}

func (h *AgentWSHandler) Connect(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		JSONUnauthorized(c)
		return
	}
	token := strings.TrimPrefix(header, "Bearer ")
	claims, err := h.Tokens.Parse(token)
	if err != nil || claims.Kind != auth.TokenAgent || claims.ServerID == "" {
		JSONUnauthorized(c)
		return
	}
	if err := h.Servers.ValidateAgentToken(c.Request.Context(), claims.ServerID, token); err != nil {
		JSONUnauthorized(c)
		return
	}

	conn, err := agentUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	agentConn := h.Hub.Register(claims.ServerID, conn)
	hostname := c.GetHeader("X-Agent-Hostname")
	version := c.GetHeader("X-Agent-Version")
	_ = h.Servers.AgentConnected(c.Request.Context(), claims.ServerID, hostname, version)
	defer func() {
		_ = h.Servers.AgentDisconnected(c.Request.Context(), claims.ServerID)
		_ = conn.Close()
	}()
	h.Hub.ReadLoop(agentConn)
}
