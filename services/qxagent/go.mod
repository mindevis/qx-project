module github.com/qxproject/qx/services/qxagent

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/qxproject/qx/pkg/log v0.0.0
	github.com/qxproject/qx/pkg/protocol v0.0.0
)

replace github.com/qxproject/qx/pkg/log => ../../pkg/log

replace github.com/qxproject/qx/pkg/protocol => ../../pkg/protocol
