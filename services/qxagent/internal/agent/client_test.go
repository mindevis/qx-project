package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestWSURLFromAPI(t *testing.T) {
	cases := map[string]string{
		"http://localhost:3000/api/v1":  "ws://localhost:3000/agent/v1/connect",
		"https://api.example.com/api/v1": "wss://api.example.com/agent/v1/connect",
		"http://localhost:3000":          "ws://localhost:3000/agent/v1/connect",
	}
	for in, want := range cases {
		if got := WSURLFromAPI(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestProcessRunnerDryRun(t *testing.T) {
	r := &ProcessRunner{DryRun: true}
	pid, err := r.Start(protocol.ServerStartPayload{JarPath: "/tmp/server.jar"})
	if err != nil || pid == 0 {
		t.Fatalf("start: pid=%d err=%v", pid, err)
	}
	exit, err := r.Stop(true, 0)
	if err != nil || exit != 0 {
		t.Fatalf("stop: exit=%d err=%v", exit, err)
	}
}

func TestProcessRunnerStartRequiresJar(t *testing.T) {
	r := &ProcessRunner{DryRun: false}
	_, err := r.Start(protocol.ServerStartPayload{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultHostname(t *testing.T) {
	if DefaultHostname() == "" {
		t.Fatal("empty hostname")
	}
}

func TestClientRunMissingToken(t *testing.T) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	c := NewClient(Config{WSURL: "ws://127.0.0.1:1", Token: "bad"})
	err := c.connectOnce(ctx)
	if err == nil {
		t.Fatal("expected dial error")
	}
}

func TestReadLoopReplaysCachedResult(t *testing.T) {
	reqID := "550e8400-e29b-41d4-a716-446655440000"
	var mu sync.Mutex
	var results []protocol.Envelope

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		payload, _ := json.Marshal(protocol.ServerStartPayload{JarPath: "/tmp/server.jar"})
		env := protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeCmdServerStart,
			RequestID: reqID,
			Payload:   payload,
		}
		data, _ := json.Marshal(env)
		_ = conn.WriteMessage(websocket.TextMessage, data)
		_ = conn.WriteMessage(websocket.TextMessage, data)

		deadline := time.Now().Add(time.Second)
		for len(results) < 2 && time.Now().Before(deadline) {
			_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, msg, err := conn.ReadMessage()
			if err != nil {
				continue
			}
			var res protocol.Envelope
			if json.Unmarshal(msg, &res) == nil && res.Type == protocol.TypeResServerStart {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	c := NewClient(Config{DryRun: true})
	if err := c.readLoop(conn); err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		// readLoop exits when peer closes.
	}

	mu.Lock()
	n := len(results)
	first := results
	mu.Unlock()
	if n != 2 {
		t.Fatalf("expected 2 cached start results, got %d", n)
	}
	if first[0].RequestID != reqID || first[1].RequestID != reqID {
		t.Fatalf("request_id mismatch")
	}
}
