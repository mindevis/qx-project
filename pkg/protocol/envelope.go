package protocol

import "encoding/json"

const Version = 1

// MaxContentFileBytes is the max jar/zip copied between game server and launcher.
// Catalog installs stream from the vendor URL with no cap, so the server can
// already hold mods larger than the old 32 MiB in-memory limit.
const MaxContentFileBytes = 256 * 1024 * 1024

const (
	TypeCmdServerInstall                 = "cmd.server.install"
	TypeCmdServerWipe                    = "cmd.server.wipe"
	TypeCmdServerCopy                    = "cmd.server.copy"
	TypeCmdServerConfigure               = "cmd.server.configure"
	TypeCmdServerStart                   = "cmd.server.start"
	TypeCmdServerStop                    = "cmd.server.stop"
	TypeCmdServerRestart                 = "cmd.server.restart"
	TypeCmdAgentPing                     = "cmd.agent.ping"
	TypeCmdConsoleInput                  = "cmd.console.input"
	TypeCmdConsoleAttach                 = "cmd.console.attach"
	TypeCmdServerPropertiesGet           = "cmd.server.properties.get"
	TypeCmdServerPropertiesPatch         = "cmd.server.properties.patch"
	TypeCmdServerFilesList               = "cmd.server.files.list"
	TypeCmdServerFilesRead               = "cmd.server.files.read"
	TypeCmdServerFilesWrite              = "cmd.server.files.write"
	TypeCmdServerFilesMkdir              = "cmd.server.files.mkdir"
	TypeCmdServerFilesDelete             = "cmd.server.files.delete"
	TypeCmdServerModsList                = "cmd.server.mods.list"
	TypeCmdServerClientModsList          = "cmd.server.clientmods.list"
	TypeCmdServerResourcepacksList       = "cmd.server.resourcepacks.list"
	TypeCmdServerClientResourcepacksList = "cmd.server.clientresourcepacks.list"
	TypeCmdServerShadersList             = "cmd.server.shaders.list"
	TypeCmdServerClientShadersList       = "cmd.server.clientshaders.list"
	TypeCmdServerPluginsList             = "cmd.server.plugins.list"
	TypeCmdServerDatapacksList           = "cmd.server.datapacks.list"
	TypeCmdServerContentInstall          = "cmd.server.content.install"
	TypeCmdServerContentUpload           = "cmd.server.content.upload"
	TypeCmdServerContentRead             = "cmd.server.content.read"
	TypeCmdServerContentDelete           = "cmd.server.content.delete"
	TypeCmdServerContentSetEnabled       = "cmd.server.content.set_enabled"
	TypeCmdInstanceFilesList             = "cmd.instance.files.list"
	TypeCmdInstanceFilesRead             = "cmd.instance.files.read"
	TypeCmdInstanceFilesWrite            = "cmd.instance.files.write"
	TypeCmdOllamaInstall                 = "cmd.ollama.install"
	TypeCmdOllamaStart                   = "cmd.ollama.start"
	TypeCmdOllamaStop                    = "cmd.ollama.stop"
	TypeCmdOllamaStatus                  = "cmd.ollama.status"
	TypeCmdOllamaModelList               = "cmd.ollama.model.list"
	TypeCmdOllamaModelPull               = "cmd.ollama.model.pull"
	TypeCmdOllamaModelDelete             = "cmd.ollama.model.delete"
	TypeCmdMySQLInstall                  = "cmd.mysql.install"
	TypeCmdMySQLUninstall                = "cmd.mysql.uninstall"
	TypeCmdMySQLStart                    = "cmd.mysql.start"
	TypeCmdMySQLStop                     = "cmd.mysql.stop"
	TypeCmdMySQLStatus                   = "cmd.mysql.status"
	TypeCmdMySQLDatabaseCreate           = "cmd.mysql.database.create"
	TypeCmdMySQLDatabaseDrop             = "cmd.mysql.database.drop"
	TypeCmdMySQLUserCreate               = "cmd.mysql.user.create"
	TypeCmdMySQLUserDrop                 = "cmd.mysql.user.drop"
	TypeCmdMySQLUserGrant                = "cmd.mysql.user.grant"

	TypeEvtAgentHeartbeat = "evt.agent.heartbeat"
	TypeEvtConsoleOutput  = "evt.console.output"
	TypeEvtServerStatus   = "evt.server.status"

	TypeResServerInstall                 = "res.server.install"
	TypeResServerWipe                    = "res.server.wipe"
	TypeResServerCopy                    = "res.server.copy"
	TypeResServerConfigure               = "res.server.configure"
	TypeResServerStart                   = "res.server.start"
	TypeResServerStop                    = "res.server.stop"
	TypeResServerPropertiesGet           = "res.server.properties.get"
	TypeResServerPropertiesPatch         = "res.server.properties.patch"
	TypeResServerFilesList               = "res.server.files.list"
	TypeResServerFilesRead               = "res.server.files.read"
	TypeResServerFilesWrite              = "res.server.files.write"
	TypeResServerFilesMkdir              = "res.server.files.mkdir"
	TypeResServerFilesDelete             = "res.server.files.delete"
	TypeResServerModsList                = "res.server.mods.list"
	TypeResServerClientModsList          = "res.server.clientmods.list"
	TypeResServerResourcepacksList       = "res.server.resourcepacks.list"
	TypeResServerClientResourcepacksList = "res.server.clientresourcepacks.list"
	TypeResServerShadersList             = "res.server.shaders.list"
	TypeResServerClientShadersList       = "res.server.clientshaders.list"
	TypeResServerPluginsList             = "res.server.plugins.list"
	TypeResServerDatapacksList           = "res.server.datapacks.list"
	TypeResServerContentInstall          = "res.server.content.install"
	TypeResServerContentUpload           = "res.server.content.upload"
	TypeResServerContentRead             = "res.server.content.read"
	TypeResServerContentDelete           = "res.server.content.delete"
	TypeResServerContentSetEnabled       = "res.server.content.set_enabled"
	TypeResInstanceFilesList             = "res.instance.files.list"
	TypeResInstanceFilesRead             = "res.instance.files.read"
	TypeResInstanceFilesWrite            = "res.instance.files.write"
	TypeResOllamaInstall                 = "res.ollama.install"
	TypeResOllamaStart                   = "res.ollama.start"
	TypeResOllamaStop                    = "res.ollama.stop"
	TypeResOllamaStatus                  = "res.ollama.status"
	TypeResOllamaModelList               = "res.ollama.model.list"
	TypeResOllamaModelPull               = "res.ollama.model.pull"
	TypeResOllamaModelDelete             = "res.ollama.model.delete"
	TypeResMySQLInstall                  = "res.mysql.install"
	TypeResMySQLUninstall                = "res.mysql.uninstall"
	TypeResMySQLStart                    = "res.mysql.start"
	TypeResMySQLStop                     = "res.mysql.stop"
	TypeResMySQLStatus                   = "res.mysql.status"
	TypeResMySQLDatabaseCreate           = "res.mysql.database.create"
	TypeResMySQLDatabaseDrop             = "res.mysql.database.drop"
	TypeResMySQLUserCreate               = "res.mysql.user.create"
	TypeResMySQLUserDrop                 = "res.mysql.user.drop"
	TypeResMySQLUserGrant                = "res.mysql.user.grant"
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
	MCVersion    string   `json:"mc_version,omitempty"`
}

type ServerStopPayload struct {
	Graceful     bool   `json:"graceful"`
	TimeoutSec   int    `json:"timeout_sec"`
	GameServerID string `json:"game_server_id,omitempty"`
	WorkDir      string `json:"work_dir,omitempty"`
}

type ServerStartResult struct {
	PID int `json:"pid"`
}

type ServerStopResult struct {
	ExitCode int `json:"exit_code"`
}

const (
	ServerStatusRunning = "running"
	ServerStatusStopped = "stopped"
	ServerStatusCrashed = "crashed"
)

type ServerStatusPayload struct {
	GameServerID string `json:"game_server_id,omitempty"`
	Status       string `json:"status"`
	PID          int    `json:"pid,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	Message      string `json:"message,omitempty"`
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
	Line         string `json:"line"`
	GameServerID string `json:"game_server_id,omitempty"`
}

type ConsoleAttachPayload struct {
	GameServerID string `json:"game_server_id,omitempty"`
	WorkDir      string `json:"work_dir"`
}

type GameServerWorkDirPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	// Remove deletes the instance directory itself. Wipe keeps an empty folder.
	Remove bool `json:"remove,omitempty"`
}

type GameServerCopyPayload struct {
	GameServerID  string `json:"game_server_id"`
	SourceWorkDir string `json:"source_work_dir"`
	DestWorkDir   string `json:"dest_work_dir"`
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
	Content      string `json:"content,omitempty"`
	ContentB64   string `json:"content_b64,omitempty"`
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

type ServerContentInstallPayload struct {
	GameServerID    string `json:"game_server_id"`
	WorkDir         string `json:"work_dir"`
	ServerType      string `json:"server_type,omitempty"`
	ContentKind     string `json:"content_kind"`
	ModTarget       string `json:"mod_target,omitempty"`
	Filename        string `json:"filename"`
	DownloadURL     string `json:"download_url"`
	AllowCustomHost bool   `json:"allow_custom_host,omitempty"`
}

type ServerContentInstallResult struct {
	Status   string `json:"status"`
	RelPath  string `json:"rel_path,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type InstanceFilesPathPayload struct {
	InstanceID string `json:"instance_id"`
	Path       string `json:"path"`
}

type InstanceFilesWritePayload struct {
	InstanceID string `json:"instance_id"`
	Path       string `json:"path"`
	Content    string `json:"content"`
}

type InstanceFilesListResult struct {
	Entries []FileEntry `json:"entries"`
}

type InstanceFilesReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

type ServerContentUploadPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	ServerType   string `json:"server_type,omitempty"`
	ContentKind  string `json:"content_kind"`
	ModTarget    string `json:"mod_target,omitempty"`
	Filename     string `json:"filename"`
	ContentB64   string `json:"content_b64"`
}

type ServerContentReadPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	ServerType   string `json:"server_type,omitempty"`
	ContentKind  string `json:"content_kind"`
	ModTarget    string `json:"mod_target,omitempty"`
	Filename     string `json:"filename"`
}

type ServerContentReadResult struct {
	Status     string `json:"status"`
	RelPath    string `json:"rel_path,omitempty"`
	Filename   string `json:"filename,omitempty"`
	ContentB64 string `json:"content_b64,omitempty"`
	Size       int64  `json:"size,omitempty"`
}

type ServerContentDeletePayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	ServerType   string `json:"server_type,omitempty"`
	ContentKind  string `json:"content_kind"`
	ModTarget    string `json:"mod_target,omitempty"`
	Filename     string `json:"filename"`
}

type ServerContentDeleteResult struct {
	Status   string `json:"status"`
	RelPath  string `json:"rel_path,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type ServerContentSetEnabledPayload struct {
	GameServerID string `json:"game_server_id"`
	WorkDir      string `json:"work_dir"`
	ServerType   string `json:"server_type,omitempty"`
	ContentKind  string `json:"content_kind"`
	ModTarget    string `json:"mod_target,omitempty"`
	Filename     string `json:"filename"`
	Enabled      bool   `json:"enabled"`
}

type ServerContentSetEnabledResult struct {
	Status   string `json:"status"`
	RelPath  string `json:"rel_path,omitempty"`
	Filename string `json:"filename,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type ServerContentUploadResult struct {
	Status   string `json:"status"`
	RelPath  string `json:"rel_path,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type OllamaInstallPayload struct {
	RootDir string `json:"root_dir,omitempty"`
}

type OllamaInstallResult struct {
	Version    string `json:"version,omitempty"`
	BinPath    string `json:"bin_path"`
	RootDir    string `json:"root_dir"`
	ModelsDir  string `json:"models_dir"`
	ListenAddr string `json:"listen_addr"`
}

type OllamaControlPayload struct {
	RootDir string `json:"root_dir,omitempty"`
}

type OllamaStatusResult struct {
	Installed  bool   `json:"installed"`
	Running    bool   `json:"running"`
	Version    string `json:"version,omitempty"`
	BinPath    string `json:"bin_path,omitempty"`
	RootDir    string `json:"root_dir,omitempty"`
	ModelsDir  string `json:"models_dir,omitempty"`
	ListenAddr string `json:"listen_addr,omitempty"`
}

type OllamaModelNamePayload struct {
	Name string `json:"name"`
}

type OllamaModel struct {
	Name       string `json:"name"`
	Size       int64  `json:"size,omitempty"`
	Digest     string `json:"digest,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
}

type OllamaModelListResult struct {
	Models []OllamaModel `json:"models"`
}

type MySQLInstallPayload struct {
	Engine       string `json:"engine"`
	Version      string `json:"version"`
	Method       string `json:"method"`
	BindAddr     string `json:"bind_addr"`
	Port         int    `json:"port"`
	RootPassword string `json:"root_password"`
}

type MySQLInstallResult struct {
	Engine   string `json:"engine"`
	Version  string `json:"version"`
	Method   string `json:"method"`
	BindAddr string `json:"bind_addr"`
	Port     int    `json:"port"`
	Image    string `json:"image,omitempty"`
}

type MySQLStatusResult struct {
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Engine    string `json:"engine,omitempty"`
	Version   string `json:"version,omitempty"`
	Method    string `json:"method,omitempty"`
	BindAddr  string `json:"bind_addr,omitempty"`
	Port      int    `json:"port,omitempty"`
	Image     string `json:"image,omitempty"`
}

type MySQLIdentPayload struct {
	Name string `json:"name"`
}

type MySQLUserPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host,omitempty"`
}

type MySQLGrantPayload struct {
	Username   string   `json:"username"`
	Host       string   `json:"host,omitempty"`
	Database   string   `json:"database"`
	Privileges []string `json:"privileges"`
}
