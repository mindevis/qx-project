package log

// Direction values for request/message logging.
const (
	DirectionIn  = "in"
	DirectionOut = "out"
)

// Transport values describe the protocol used.
const (
	TransportHTTP    = "http"
	TransportAgentWS = "agent-ws"
	TransportSSH     = "ssh"
)
