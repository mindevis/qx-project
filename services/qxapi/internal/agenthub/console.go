package agenthub

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/pkg/protocol"
)

type consoleSubscriber struct {
	conn         *websocket.Conn
	sendMu       sync.Mutex
	gameServerID string
}

// ConsolePanelMessage is the JSON shape sent to web console clients.
type ConsolePanelMessage struct {
	Type         string `json:"type"`
	Stream       string `json:"stream,omitempty"`
	Line         string `json:"line,omitempty"`
	GameServerID string `json:"game_server_id,omitempty"`
}

func (h *Hub) SubscribeConsole(serverID string, conn *websocket.Conn, gameServerID string) {
	h.mu.Lock()
	if h.consoleSubs == nil {
		h.consoleSubs = make(map[string]map[*websocket.Conn]*consoleSubscriber)
	}
	if h.consoleSubs[serverID] == nil {
		h.consoleSubs[serverID] = make(map[*websocket.Conn]*consoleSubscriber)
	}
	h.consoleSubs[serverID][conn] = &consoleSubscriber{conn: conn, gameServerID: strings.TrimSpace(gameServerID)}
	h.mu.Unlock()
	h.replayConsoleHistory(serverID, conn)
}

func (h *Hub) UnsubscribeConsole(serverID string, conn *websocket.Conn) {
	h.mu.Lock()
	if subs, ok := h.consoleSubs[serverID]; ok {
		delete(subs, conn)
		if len(subs) == 0 {
			delete(h.consoleSubs, serverID)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) BroadcastConsole(serverID string, payload protocol.ConsoleOutputPayload) {
	msg := ConsolePanelMessage{
		Type:         "output",
		Stream:       payload.Stream,
		Line:         payload.Line,
		GameServerID: payload.GameServerID,
	}
	h.appendConsoleHistory(serverID, msg)
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	subs := h.consoleSubs[serverID]
	clients := make([]*consoleSubscriber, 0, len(subs))
	for _, sub := range subs {
		clients = append(clients, sub)
	}
	h.mu.RUnlock()

	for _, sub := range clients {
		if !consoleLineMatches(sub.gameServerID, payload.GameServerID) {
			continue
		}
		_ = h.writeConsoleLocked(sub, data)
	}
}

func (h *Hub) appendConsoleHistory(serverID string, msg ConsolePanelMessage) {
	if msg.Line == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.consoleHistory == nil {
		h.consoleHistory = make(map[string][]ConsolePanelMessage)
	}
	history := append(h.consoleHistory[serverID], msg)
	if len(history) > consoleHistoryLimit {
		history = history[len(history)-consoleHistoryLimit:]
	}
	h.consoleHistory[serverID] = history
}

func (h *Hub) replayConsoleHistory(serverID string, conn *websocket.Conn) {
	h.mu.RLock()
	history := append([]ConsolePanelMessage(nil), h.consoleHistory[serverID]...)
	var sub *consoleSubscriber
	if subs := h.consoleSubs[serverID]; subs != nil {
		sub = subs[conn]
	}
	h.mu.RUnlock()

	for _, msg := range history {
		if sub != nil && !consoleLineMatches(sub.gameServerID, msg.GameServerID) {
			continue
		}
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if sub != nil {
			_ = h.writeConsoleLocked(sub, data)
		} else {
			_ = conn.WriteMessage(websocket.TextMessage, data)
		}
	}
}

// WriteConsolePanel sends a JSON message to a console panel client using the same
// write lock as BroadcastConsole (gorilla/websocket allows only one writer at a time).
func (h *Hub) WriteConsolePanel(serverID string, conn *websocket.Conn, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.mu.RLock()
	var sub *consoleSubscriber
	if subs := h.consoleSubs[serverID]; subs != nil {
		sub = subs[conn]
	}
	h.mu.RUnlock()
	if sub == nil {
		return conn.WriteMessage(websocket.TextMessage, data)
	}
	return h.writeConsoleLocked(sub, data)
}

func (h *Hub) writeConsoleLocked(sub *consoleSubscriber, data []byte) error {
	sub.sendMu.Lock()
	defer sub.sendMu.Unlock()
	return sub.conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Hub) SendConsoleInput(ctx context.Context, serverID, line, gameServerID string) error {
	payload, err := json.Marshal(protocol.ConsoleInputPayload{
		Line:         line,
		GameServerID: strings.TrimSpace(gameServerID),
	})
	if err != nil {
		return err
	}
	return h.SendCommand(ctx, serverID, protocol.Envelope{
		Type:    protocol.TypeCmdConsoleInput,
		Payload: payload,
	})
}

func consoleLineMatches(filterGameServerID, lineGameServerID string) bool {
	if filterGameServerID == "" {
		return true
	}
	return lineGameServerID == filterGameServerID
}
