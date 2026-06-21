package log

import "testing"

func TestDirectionConstants(t *testing.T) {
	if DirectionIn != "in" || DirectionOut != "out" {
		t.Fatalf("unexpected direction constants")
	}
	if TransportHTTP != "http" || TransportAgentWS != "agent-ws" || TransportSSH != "ssh" {
		t.Fatalf("unexpected transport constants")
	}
}
