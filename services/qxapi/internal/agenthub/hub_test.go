package agenthub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/qxproject/qx/pkg/protocol"
)

func TestHubRegisterAndOnline(t *testing.T) {
	h := New(nil)
	server := wsTestServer(t, func(*websocket.Conn) {})
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if h.IsOnline("srv-1") {
		t.Fatal("expected offline")
	}
	agentConn := h.Register("srv-1", conn)
	if !h.IsOnline("srv-1") {
		t.Fatal("expected online")
	}
	h.Unregister("srv-1", agentConn)
	if h.IsOnline("srv-1") {
		t.Fatal("expected offline after unregister")
	}
}

func TestSendCommandOffline(t *testing.T) {
	h := New(nil)
	err := h.SendCommand(context.Background(), "missing", protocol.Envelope{Type: protocol.TypeCmdServerStart})
	if err != ErrAgentOffline {
		t.Fatalf("offline: %v", err)
	}
}

func TestSendCommandWritesMessage(t *testing.T) {
	h := New(nil)
	received := make(chan protocol.Envelope, 1)
	server := wsTestServer(t, func(c *websocket.Conn) {
		_, data, err := c.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) == nil {
			received <- env
		}
	})
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	h.Register("srv-1", conn)

	payload, _ := json.Marshal(protocol.ServerStopPayload{Graceful: true, TimeoutSec: 5})
	err = h.SendCommand(context.Background(), "srv-1", protocol.Envelope{
		Type:    protocol.TypeCmdServerStop,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	select {
	case env := <-received:
		if env.Type != protocol.TypeCmdServerStop || env.V != protocol.Version || env.RequestID == "" {
			t.Fatalf("env: %+v", env)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestReadLoopOnEvent(t *testing.T) {
	events := make(chan protocol.Envelope, 1)
	h := New(func(_ string, env protocol.Envelope) {
		events <- env
	})

	server := wsTestServer(t, func(c *websocket.Conn) {
		payload, _ := json.Marshal(protocol.HeartbeatPayload{CPUPercent: 1.5})
		_ = c.WriteJSON(protocol.Envelope{
			V:    protocol.Version,
			Type: protocol.TypeEvtAgentHeartbeat,
			Payload: payload,
		})
		time.Sleep(50 * time.Millisecond)
	})
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	agentConn := h.Register("srv-1", conn)
	go h.ReadLoop(agentConn)

	select {
	case env := <-events:
		if env.Type != protocol.TypeEvtAgentHeartbeat {
			t.Fatalf("type: %s", env.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestHubReconnectReplacesConnection(t *testing.T) {
	h := New(nil)
	received := make(chan protocol.Envelope, 1)

	server := wsTestServer(t, func(c *websocket.Conn) {
		_, data, err := c.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) == nil {
			received <- env
		}
	})
	defer server.Close()

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial1: %v", err)
	}
	h.Register("srv-1", conn1)

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial2: %v", err)
	}
	h.Register("srv-1", conn2)

	if !h.IsOnline("srv-1") {
		t.Fatal("expected online after reconnect")
	}

	err = h.SendCommand(context.Background(), "srv-1", protocol.Envelope{
		Type: protocol.TypeCmdAgentPing,
	})
	if err != nil {
		t.Fatalf("send after reconnect: %v", err)
	}

	select {
	case env := <-received:
		if env.Type != protocol.TypeCmdAgentPing {
			t.Fatalf("unexpected type: %s", env.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for command on new connection")
	}

	_ = conn1.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	if _, _, err := conn1.ReadMessage(); err == nil {
		t.Fatal("expected old connection to be closed")
	}
	_ = conn2.Close()
}

func TestReadLoopUnregisterDoesNotDropReplacementConnection(t *testing.T) {
	h := New(nil)
	server := wsTestServer(t, func(c *websocket.Conn) {
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer server.Close()

	conn1, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial1: %v", err)
	}
	agentConn1 := h.Register("srv-1", conn1)
	done := make(chan struct{})
	go func() {
		h.ReadLoop(agentConn1)
		close(done)
	}()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial2: %v", err)
	}
	h.Register("srv-1", conn2)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first read loop to exit")
	}

	if !h.IsOnline("srv-1") {
		t.Fatal("replacement connection should stay registered")
	}
	_ = conn2.Close()
}

func TestSetOnEvent(t *testing.T) {
	h := New(nil)
	called := false
	h.SetOnEvent(func(string, protocol.Envelope) { called = true })
	h.mu.RLock()
	fn := h.onEvent
	h.mu.RUnlock()
	if fn == nil {
		t.Fatal("onEvent nil")
	}
	fn("id", protocol.Envelope{})
	if !called {
		t.Fatal("callback not called")
	}
}

func wsTestServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(conn)
	}))
}

func wsURL(httpURL string) string {
	return strings.Replace(httpURL, "http://", "ws://", 1)
}
