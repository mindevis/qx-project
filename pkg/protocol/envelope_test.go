package protocol

import "testing"

func TestProtocolConstants(t *testing.T) {
	if Version != 1 {
		t.Fatalf("version: %d", Version)
	}
	if TypeCmdServerStart == "" || TypeEvtAgentHeartbeat == "" {
		t.Fatal("expected command types")
	}
	if TypeCmdOllamaInstall == "" || TypeResOllamaModelPull == "" {
		t.Fatal("expected ollama command types")
	}
}
