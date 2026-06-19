package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/servers"
)

var consoleUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ServerConsoleHandler struct {
	Servers *servers.Service
	Tokens  *auth.TokenService
}

type consolePanelIn struct {
	Type string `json:"type"`
	Line string `json:"line"`
}

func (h *ServerConsoleHandler) Connect(c *gin.Context) {
	token := consoleTokenFromRequest(c)
	if token == "" {
		JSONUnauthorized(c)
		return
	}
	claims, err := h.Tokens.Parse(token)
	if err != nil || claims.Kind != auth.TokenAccess {
		JSONUnauthorized(c)
		return
	}

	serverID := c.Param("id")
	if _, err := h.Servers.Get(c.Request.Context(), claims.UserID, serverID); err != nil {
		if errors.Is(err, servers.ErrNotFound) {
			JSONError(c, http.StatusNotFound, "NOT_FOUND", "server not found")
			return
		}
		if errors.Is(err, servers.ErrForbidden) {
			JSONError(c, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}
		JSONInternal(c)
		return
	}

	conn, err := consoleUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	hub := h.Servers.Hub()
	if hub == nil {
		_ = conn.WriteJSON(agenthubConsoleStatus("error", "console unavailable"))
		return
	}

	hub.SubscribeConsole(serverID, conn)
	defer hub.UnsubscribeConsole(serverID, conn)

	_ = conn.WriteJSON(agenthubConsoleStatus("connected", ""))

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg consolePanelIn
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != "input" || strings.TrimSpace(msg.Line) == "" {
			continue
		}
		if err := h.Servers.SendConsoleInput(c.Request.Context(), claims.UserID, serverID, msg.Line); err != nil {
			if errors.Is(err, servers.ErrAgentOffline) {
				_ = conn.WriteJSON(agenthubConsoleStatus("error", "agent offline"))
			}
		}
	}
}

func consoleTokenFromRequest(c *gin.Context) string {
	if q := c.Query("access_token"); q != "" {
		return q
	}
	header := c.GetHeader("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

func agenthubConsoleStatus(status, detail string) map[string]string {
	out := map[string]string{"type": "status", "status": status}
	if detail != "" {
		out["detail"] = detail
	}
	return out
}
