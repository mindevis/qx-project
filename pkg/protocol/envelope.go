package protocol

import "encoding/json"

const Version = 1

const (
	TypeCmdServerStart   = "cmd.server.start"
	TypeCmdServerStop    = "cmd.server.stop"
	TypeCmdServerRestart = "cmd.server.restart"
	TypeCmdAgentPing     = "cmd.agent.ping"
	TypeCmdConsoleInput  = "cmd.console.input"

	TypeEvtAgentHeartbeat = "evt.agent.heartbeat"
	TypeEvtConsoleOutput  = "evt.console.output"

	TypeResServerStart = "res.server.start"
	TypeResServerStop  = "res.server.stop"
)

type Envelope struct {
	V         int             `json:"v"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	TS        string          `json:"ts"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type ServerStartPayload struct {
	ServerType string   `json:"server_type"`
	JarPath    string   `json:"jar_path"`
	JVMArgs    []string `json:"jvm_args"`
	ExtraArgs  []string `json:"extra_args"`
}

type ServerStopPayload struct {
	Graceful   bool `json:"graceful"`
	TimeoutSec int  `json:"timeout_sec"`
}

type ServerStartResult struct {
	PID int `json:"pid"`
}

type ServerStopResult struct {
	ExitCode int `json:"exit_code"`
}

type HeartbeatPayload struct {
	CPUPercent float64 `json:"cpu_percent,omitempty"`
	MemMB      int     `json:"mem_mb,omitempty"`
}

type ConsoleOutputPayload struct {
	Stream string `json:"stream"`
	Line   string `json:"line"`
}

type ConsoleInputPayload struct {
	Line string `json:"line"`
}
