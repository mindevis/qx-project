package agenthub

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/pkg/protocol"
)

type consoleSubscriber struct {
	conn   *websocket.Conn
	sendMu sync.Mutex
}

// ConsolePanelMessage is the JSON shape sent to web console clients.
type ConsolePanelMessage struct {
	Type   string `json:"type"`
	Stream string `json:"stream,omitempty"`
	Line   string `json:"line,omitempty"`
}

func (h *Hub) SubscribeConsole(serverID string, conn *websocket.Conn) {
	h.mu.Lock()
	if h.consoleSubs == nil {
		h.consoleSubs = make(map[string]map[*websocket.Conn]*consoleSubscriber)
	}
	if h.consoleSubs[serverID] == nil {
		h.consoleSubs[serverID] = make(map[*websocket.Conn]*consoleSubscriber)
	}
	h.consoleSubs[serverID][conn] = &consoleSubscriber{conn: conn}
	h.mu.Unlock()
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
	msg := ConsolePanelMessage{Type: "output", Stream: payload.Stream, Line: payload.Line}
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
		sub.sendMu.Lock()
		_ = sub.conn.WriteMessage(websocket.TextMessage, data)
		sub.sendMu.Unlock()
	}
}

func (h *Hub) SendConsoleInput(ctx context.Context, serverID, line string) error {
	payload, err := json.Marshal(protocol.ConsoleInputPayload{Line: line})
	if err != nil {
		return err
	}
	return h.SendCommand(ctx, serverID, protocol.Envelope{
		Type:    protocol.TypeCmdConsoleInput,
		Payload: payload,
	})
}
