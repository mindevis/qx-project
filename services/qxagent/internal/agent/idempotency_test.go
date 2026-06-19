package agent

import (
	"testing"

	"github.com/qxproject/qx/pkg/protocol"
)

func TestRequestCacheSetGetAndEvict(t *testing.T) {
	cache := newRequestCache(2)
	cache.Set("a", protocol.Envelope{RequestID: "a", Type: protocol.TypeResServerStart})
	cache.Set("b", protocol.Envelope{RequestID: "b", Type: protocol.TypeResServerStop})
	cache.Set("c", protocol.Envelope{RequestID: "c", Type: protocol.TypeEvtAgentHeartbeat})

	if _, ok := cache.Get("c"); !ok {
		t.Fatal("expected c")
	}
	if _, ok := cache.Get("a"); ok {
		t.Fatal("expected oldest entry a evicted")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Fatal("expected b retained")
	}
}
