module github.com/qxproject/qx/services/qxagent

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/pelletier/go-toml/v2 v2.4.0
	github.com/qxproject/qx/pkg/log v0.0.0
	github.com/qxproject/qx/pkg/mcproxy v0.0.0
	github.com/qxproject/qx/pkg/mojangjava v0.0.0
	github.com/qxproject/qx/pkg/mysqlutil v0.0.0
	github.com/qxproject/qx/pkg/protocol v0.0.0
	github.com/qxproject/qx/pkg/reporoot v0.0.0
	github.com/qxproject/qx/pkg/safepath v0.0.0
)

require github.com/klauspost/compress v1.18.6 // indirect

replace github.com/qxproject/qx/pkg/reporoot => ../../pkg/reporoot

replace github.com/qxproject/qx/pkg/log => ../../pkg/log

replace github.com/qxproject/qx/pkg/mojangjava => ../../pkg/mojangjava

replace github.com/qxproject/qx/pkg/mysqlutil => ../../pkg/mysqlutil

replace github.com/qxproject/qx/pkg/protocol => ../../pkg/protocol

replace github.com/qxproject/qx/pkg/safepath => ../../pkg/safepath

replace github.com/qxproject/qx/pkg/mcproxy => ../../pkg/mcproxy
