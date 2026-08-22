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

func TestBroadcastConsoleToSubscriber(t *testing.T) {
	h := New(nil)
	ready := make(chan struct{})
	server := wsTestServer(t, func(c *websocket.Conn) {
		h.SubscribeConsole("srv-1", c, "")
		close(ready)
		go func() {
			time.Sleep(20 * time.Millisecond)
			h.BroadcastConsole("srv-1", protocol.ConsoleOutputPayload{Stream: "stdout", Line: "hello"})
		}()
	})
	defer server.Close()

	panelConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial panel: %v", err)
	}
	defer panelConn.Close()
	<-ready

	_ = panelConn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := panelConn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg ConsolePanelMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Type != "output" || msg.Line != "hello" {
		t.Fatalf("msg: %s err=%v", data, err)
	}
}

func TestConsoleHistoryReplay(t *testing.T) {
	h := New(nil)
	h.BroadcastConsole("srv-1", protocol.ConsoleOutputPayload{Stream: "stdout", Line: "before subscribe"})

	server := wsTestServer(t, func(c *websocket.Conn) {
		h.SubscribeConsole("srv-1", c, "")
		time.Sleep(200 * time.Millisecond)
	})
	defer server.Close()

	panelConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial panel: %v", err)
	}
	defer panelConn.Close()

	_ = panelConn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := panelConn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg ConsolePanelMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Line != "before subscribe" {
		t.Fatalf("replay msg: %s err=%v", data, err)
	}
}

func TestSendConsoleInput(t *testing.T) {
	h := New(nil)
	received := make(chan protocol.Envelope, 1)
	agentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var env protocol.Envelope
		if json.Unmarshal(data, &env) == nil {
			received <- env
		}
	}))
	defer agentServer.Close()

	conn, _, err := websocket.DefaultDialer.Dial(strings.Replace(agentServer.URL, "http://", "ws://", 1), nil)
	if err != nil {
		t.Fatalf("dial agent: %v", err)
	}
	defer conn.Close()
	h.Register("srv-1", conn)

	if err := h.SendConsoleInput(context.Background(), "srv-1", "say hi", ""); err != nil {
		t.Fatalf("send input: %v", err)
	}

	select {
	case env := <-received:
		if env.Type != protocol.TypeCmdConsoleInput {
			t.Fatalf("type: %s", env.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestUnsubscribeConsole(t *testing.T) {
	h := New(nil)
	server := wsTestServer(t, func(*websocket.Conn) {})
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	h.SubscribeConsole("srv-1", conn, "")
	h.UnsubscribeConsole("srv-1", conn)
	h.BroadcastConsole("srv-1", protocol.ConsoleOutputPayload{Line: "x"})
}

func TestBroadcastConsoleFiltersGameServer(t *testing.T) {
	h := New(nil)
	ready := make(chan struct{})
	server := wsTestServer(t, func(c *websocket.Conn) {
		h.SubscribeConsole("srv-1", c, "gs-1")
		close(ready)
		go func() {
			time.Sleep(20 * time.Millisecond)
			h.BroadcastConsole("srv-1", protocol.ConsoleOutputPayload{Stream: "stdout", Line: "other", GameServerID: "gs-2"})
			h.BroadcastConsole("srv-1", protocol.ConsoleOutputPayload{Stream: "stdout", Line: "untagged"})
			h.BroadcastConsole("srv-1", protocol.ConsoleOutputPayload{Stream: "stdout", Line: "mine", GameServerID: "gs-1"})
		}()
	})
	defer server.Close()

	panelConn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err != nil {
		t.Fatalf("dial panel: %v", err)
	}
	defer panelConn.Close()
	<-ready

	_ = panelConn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := panelConn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg ConsolePanelMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Line != "mine" {
		t.Fatalf("expected filtered line mine, got %s err=%v", data, err)
	}
}
