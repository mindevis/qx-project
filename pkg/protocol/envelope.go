package protocol

import "encoding/json"

const Version = 1

const (
	TypeCmdServerInstall   = "cmd.server.install"
	TypeCmdServerConfigure = "cmd.server.configure"
	TypeCmdServerStart     = "cmd.server.start"
	TypeCmdServerStop    = "cmd.server.stop"
	TypeCmdServerRestart = "cmd.server.restart"
	TypeCmdAgentPing     = "cmd.agent.ping"
	TypeCmdConsoleInput          = "cmd.console.input"
	TypeCmdConsoleAttach         = "cmd.console.attach"
	TypeCmdServerPropertiesGet   = "cmd.server.properties.get"
	TypeCmdServerPropertiesPatch = "cmd.server.properties.patch"
	TypeCmdServerFilesList       = "cmd.server.files.list"
	TypeCmdServerFilesRead       = "cmd.server.files.read"
	TypeCmdServerFilesWrite      = "cmd.server.files.write"
	TypeCmdServerModsList        = "cmd.server.mods.list"

	TypeEvtAgentHeartbeat = "evt.agent.heartbeat"
	TypeEvtConsoleOutput  = "evt.console.output"

	TypeResServerInstall          = "res.server.install"
	TypeResServerConfigure        = "res.server.configure"
	TypeResServerStart            = "res.server.start"
	TypeResServerStop             = "res.server.stop"
	TypeResServerPropertiesGet    = "res.server.properties.get"
	TypeResServerPropertiesPatch  = "res.server.properties.patch"
	TypeResServerFilesList        = "res.server.files.list"
	TypeResServerFilesRead        = "res.server.files.read"
	TypeResServerFilesWrite       = "res.server.files.write"
	TypeResServerModsList         = "res.server.mods.list"
)

type Envelope struct {
	V         int             `json:"v"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	TS        string          `json:"ts"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type ServerInstallPayload struct {
	GameServerID  string `json:"game_server_id"`
	ServerType    string `json:"server_type"`
	MCVersion     string `json:"mc_version"`
	LoaderVersion string `json:"loader_version,omitempty"`
	WorkDir       string `json:"work_dir"`
	Name          string `json:"name"`
	Address       string `json:"address,omitempty"`
	Port          int    `json:"port"`
	RconPassword  string `json:"rcon_password"`
}

type ServerConfigurePayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	Name         string `json:"name"`
	Address      string `json:"address,omitempty"`
	Port         int    `json:"port"`
	RconPassword string `json:"rcon_password"`
}

type ServerInstallResult struct {
	WorkDir   string   `json:"work_dir"`
	JarPath   string   `json:"jar_path,omitempty"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	JVMArgs   []string `json:"jvm_args,omitempty"`
	ExtraArgs []string `json:"extra_args,omitempty"`
	JavaBin   string   `json:"java_bin,omitempty"`
}

type ServerStartPayload struct {
	GameServerID string   `json:"game_server_id,omitempty"`
	ServerType   string   `json:"server_type"`
	JarPath      string   `json:"jar_path"`
	WorkDir      string   `json:"work_dir,omitempty"`
	Command      string   `json:"command,omitempty"`
	Args         []string `json:"args,omitempty"`
	JVMArgs      []string `json:"jvm_args"`
	ExtraArgs    []string `json:"extra_args"`
	JavaBin      string   `json:"java_bin,omitempty"`
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
	Stream       string `json:"stream"`
	Line         string `json:"line"`
	GameServerID string `json:"game_server_id,omitempty"`
}

type ConsoleInputPayload struct {
	Line string `json:"line"`
}

type ConsoleAttachPayload struct {
	GameServerID string `json:"game_server_id,omitempty"`
	WorkDir      string `json:"work_dir"`
}

type GameServerWorkDirPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
}

type ServerPropertiesPatchPayload struct {
	GameServerID string            `json:"game_server_id"`
	WorkDir      string            `json:"work_dir"`
	Updates      map[string]string `json:"updates"`
}

type PropertyEntry struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Boolean bool   `json:"boolean,omitempty"`
}

type ServerPropertiesResult struct {
	Properties []PropertyEntry `json:"properties"`
}

type ServerFilesPathPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	Path         string `json:"path"`
}

type ServerFilesWritePayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	Path         string `json:"path"`
	Content      string `json:"content"`
}

type ServerModsListPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	ServerType   string `json:"server_type"`
}

type FileEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size,omitempty"`
}

type ServerFilesListResult struct {
	Entries []FileEntry `json:"entries"`
}

type ServerFilesReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

type ServerModsListResult struct {
	Entries []FileEntry `json:"entries"`
}
