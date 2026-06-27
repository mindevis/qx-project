package agenthub

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/pkg/protocol"
)

var (
	ErrAgentOffline = errors.New("agent offline")
	ErrTimeout      = errors.New("command timeout")
)

type Conn struct {
	ServerID string
	Conn     *websocket.Conn
	sendMu   sync.Mutex
}

type Hub struct {
	mu             sync.RWMutex
	agents         map[string]*Conn
	consoleSubs    map[string]map[*websocket.Conn]*consoleSubscriber
	consoleHistory map[string][]ConsolePanelMessage
	onEvent        func(serverID string, env protocol.Envelope)
}

const consoleHistoryLimit = 500

func New(onEvent func(serverID string, env protocol.Envelope)) *Hub {
	return &Hub{
		agents:         make(map[string]*Conn),
		consoleHistory: make(map[string][]ConsolePanelMessage),
		onEvent:        onEvent,
	}
}

func (h *Hub) SetOnEvent(fn func(serverID string, env protocol.Envelope)) {
	h.mu.Lock()
	h.onEvent = fn
	h.mu.Unlock()
}

func (h *Hub) Register(serverID string, conn *websocket.Conn) *Conn {
	c := &Conn{ServerID: serverID, Conn: conn}
	h.mu.Lock()
	if old, ok := h.agents[serverID]; ok {
		_ = old.Conn.Close()
	}
	h.agents[serverID] = c
	h.mu.Unlock()
	return c
}

func (h *Hub) Unregister(serverID string) {
	h.mu.Lock()
	delete(h.agents, serverID)
	h.mu.Unlock()
}

func (h *Hub) IsOnline(serverID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[serverID]
	return ok
}

func (h *Hub) SendCommand(ctx context.Context, serverID string, env protocol.Envelope) error {
	h.mu.RLock()
	agent, ok := h.agents[serverID]
	h.mu.RUnlock()
	if !ok {
		return ErrAgentOffline
	}
	if env.RequestID == "" {
		env.RequestID = uuid.NewString()
	}
	if env.V == 0 {
		env.V = protocol.Version
	}
	if env.TS == "" {
		env.TS = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	agent.sendMu.Lock()
	err = agent.Conn.WriteMessage(websocket.TextMessage, data)
	agent.sendMu.Unlock()
	if err != nil {
		return err
	}
	slog.Debug("agent message",
		"direction", qxlog.DirectionOut,
		"transport", qxlog.TransportAgentWS,
		"server_id", serverID,
		"type", env.Type,
		"request_id", env.RequestID,
	)
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (h *Hub) ReadLoop(c *Conn) {
	defer h.Unregister(c.ServerID)
	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		if env.Type != protocol.TypeEvtAgentHeartbeat && env.Type != protocol.TypeEvtConsoleOutput {
			slog.Debug("agent message",
				"direction", qxlog.DirectionIn,
				"transport", qxlog.TransportAgentWS,
				"server_id", c.ServerID,
				"type", env.Type,
				"request_id", env.RequestID,
			)
		}
		if h.onEvent != nil {
			h.mu.RLock()
			fn := h.onEvent
			h.mu.RUnlock()
			if fn != nil {
				fn(c.ServerID, env)
			}
		}
	}
}
