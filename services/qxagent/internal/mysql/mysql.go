package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qxproject/qx/pkg/mysqlutil"
)

const (
	DefaultRoot   = "/opt/qxsystem/mysql"
	ContainerName = "qx-mysql"
	systemdMaria  = "mariadb.service"
	systemdMySQL  = "mysql.service"
)

var (
	ErrNotInstalled = errors.New("mysql is not installed")
	ErrNotRunning   = errors.New("mysql is not running")
	ErrUnsupported  = errors.New("unsupported mysql install")
)

var (
	commandContext = exec.CommandContext
	writeFileFn    = os.WriteFile
	mkdirAllFn     = os.MkdirAll
	readFileFn     = os.ReadFile
)

type Manager struct {
	RootDir    string
	DryRun     bool
	mu         sync.Mutex
	dryRunning bool
}

type InstallSpec struct {
	Engine       string
	Version      string
	Method       string
	BindAddr     string
	Port         int
	RootPassword string
}

type InstallResult struct {
	Engine   string
	Version  string
	Method   string
	BindAddr string
	Port     int
	Image    string
}

type Status struct {
	Installed bool
	Running   bool
	Engine    string
	Version   string
	Method    string
	BindAddr  string
	Port      int
	Image     string
}

type instanceConfig struct {
	Engine   string `json:"engine"`
	Version  string `json:"version"`
	Method   string `json:"method"`
	BindAddr string `json:"bind_addr"`
	Port     int    `json:"port"`
	Image    string `json:"image,omitempty"`
}

func RootFromServerRoot(serverRoot string) string {
	serverRoot = strings.TrimSpace(serverRoot)
	if serverRoot == "" {
		return DefaultRoot
	}
	return filepath.Join(filepath.Dir(serverRoot), "mysql")
}

func NewManager(rootDir string, dryRun bool) *Manager {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = DefaultRoot
	}
	return &Manager{RootDir: rootDir, DryRun: dryRun}
}

func (m *Manager) Install(ctx context.Context, spec InstallSpec, onLog func(string)) (InstallResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	engine, version, err := mysqlutil.NormalizeEngineVersion(spec.Engine, spec.Version)
	if err != nil {
		return InstallResult{}, err
	}
	method, err := mysqlutil.NormalizeMethod(spec.Method)
	if err != nil {
		return InstallResult{}, err
	}
	bind, err := mysqlutil.NormalizeBind(spec.BindAddr)
	if err != nil {
		return InstallResult{}, err
	}
	port, err := mysqlutil.NormalizePort(spec.Port)
	if err != nil {
		return InstallResult{}, err
	}
	password := strings.TrimSpace(spec.RootPassword)
	if password == "" {
		return InstallResult{}, errors.New("root password required")
	}
	image, err := mysqlutil.DockerImage(engine, version)
	if err != nil {
		return InstallResult{}, err
	}

	result := InstallResult{
		Engine:   engine,
		Version:  version,
		Method:   method,
		BindAddr: bind,
		Port:     port,
		Image:    image,
	}
	logLine(onLog, fmt.Sprintf("[QX] Installing %s %s (%s)", engine, version, method))
	if err := mkdirAllFn(m.dataDir(), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("mkdir mysql: %w", err)
	}
	if err := m.writePassword(password); err != nil {
		return InstallResult{}, err
	}
	if err := m.writeInstance(instanceConfig{
		Engine:   engine,
		Version:  version,
		Method:   method,
		BindAddr: bind,
		Port:     port,
		Image:    image,
	}); err != nil {
		return InstallResult{}, err
	}
	if m.DryRun {
		m.dryRunning = false
		logLine(onLog, "[QX] MySQL install dry-run")
		return result, nil
	}
	switch method {
	case mysqlutil.MethodDocker:
		if err := m.installDocker(ctx, result, password, onLog); err != nil {
			m.forgetInstallMarker()
			return InstallResult{}, err
		}
	case mysqlutil.MethodNative:
		if err := m.installNative(ctx, result, password, onLog); err != nil {
			m.forgetInstallMarker()
			return InstallResult{}, err
		}
	default:
		return InstallResult{}, ErrUnsupported
	}
	logLine(onLog, "[QX] MySQL installed")
	return result, nil
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, ok := m.readInstance()
	if !ok && !m.DryRun {
		return ErrNotInstalled
	}
	if m.DryRun {
		m.dryRunning = true
		return nil
	}
	switch cfg.Method {
	case mysqlutil.MethodDocker:
		if err := m.run(ctx, "docker", "start", ContainerName); err != nil {
			return err
		}
		return m.waitReady(ctx, 90*time.Second)
	default:
		if err := m.run(ctx, "systemctl", "start", m.nativeUnit(cfg.Engine)); err != nil {
			if cfg.Engine == mysqlutil.EngineMariaDB {
				_ = m.run(ctx, "systemctl", "start", systemdMySQL)
			} else {
				return err
			}
		}
		return m.waitReady(ctx, 90*time.Second)
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DryRun {
		m.dryRunning = false
		return nil
	}
	cfg, ok := m.readInstance()
	if !ok {
		return nil
	}
	if cfg.Method == mysqlutil.MethodDocker {
		_ = m.run(ctx, "docker", "stop", ContainerName)
		return nil
	}
	_ = m.run(ctx, "systemctl", "stop", m.nativeUnit(cfg.Engine))
	if cfg.Engine == mysqlutil.EngineMariaDB {
		_ = m.run(ctx, "systemctl", "stop", systemdMySQL)
	}
	return nil
}

func (m *Manager) Uninstall(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DryRun {
		m.dryRunning = false
		return m.clearLocalState()
	}
	cfg, ok := m.readInstance()
	if ok && cfg.Method == mysqlutil.MethodDocker {
		_ = m.run(ctx, "docker", "rm", "-f", ContainerName)
	} else if ok {
		_ = m.run(ctx, "systemctl", "stop", m.nativeUnit(cfg.Engine))
		_ = m.run(ctx, "systemctl", "disable", m.nativeUnit(cfg.Engine))
		if cfg.Engine == mysqlutil.EngineMariaDB {
			_ = m.run(ctx, "systemctl", "stop", systemdMySQL)
			_ = m.run(ctx, "systemctl", "disable", systemdMySQL)
		}
	} else {
		_ = m.run(ctx, "docker", "rm", "-f", ContainerName)
	}
	return m.clearLocalState()
}

func (m *Manager) forgetInstallMarker() {
	_ = os.Remove(m.instancePath())
}

func (m *Manager) clearLocalState() error {
	_ = os.Remove(m.instancePath())
	_ = os.Remove(m.passwordPath())
	_ = os.RemoveAll(m.dataDir())
	return nil
}

func (m *Manager) Status(ctx context.Context) Status {
	cfg, ok := m.readInstance()
	st := Status{
		Installed: ok,
		Engine:    cfg.Engine,
		Version:   cfg.Version,
		Method:    cfg.Method,
		BindAddr:  cfg.BindAddr,
		Port:      cfg.Port,
		Image:     cfg.Image,
	}
	if !ok {
		return st
	}
	if m.DryRun {
		st.Running = m.dryRunning
		return st
	}
	st.Running = m.ping(ctx)
	return st
}

func (m *Manager) CreateDatabase(ctx context.Context, name string) error {
	sql, err := mysqlutil.CreateDatabaseSQL(name)
	if err != nil {
		return err
	}
	return m.execSQL(ctx, sql)
}

func (m *Manager) DropDatabase(ctx context.Context, name string) error {
	sql, err := mysqlutil.DropDatabaseSQL(name)
	if err != nil {
		return err
	}
	return m.execSQL(ctx, sql)
}

func (m *Manager) CreateUser(ctx context.Context, user, host, password string) error {
	sql, err := mysqlutil.CreateUserSQL(user, host, password)
	if err != nil {
		return err
	}
	return m.execSQL(ctx, sql, "FLUSH PRIVILEGES")
}

func (m *Manager) DropUser(ctx context.Context, user, host string) error {
	sql, err := mysqlutil.DropUserSQL(user, host)
	if err != nil {
		return err
	}
	return m.execSQL(ctx, sql, "FLUSH PRIVILEGES")
}

func (m *Manager) Grant(ctx context.Context, user, host, database string, privileges []string) error {
	stmts, err := mysqlutil.ApplyGrantStatements(user, host, database, privileges)
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		if err := m.execSQL(ctx, stmt); err != nil {
			if strings.Contains(strings.ToUpper(stmt), "REVOKE") {
				continue
			}
			return err
		}
	}
	return nil
}

func (m *Manager) installDocker(ctx context.Context, spec InstallResult, password string, onLog func(string)) error {
	if err := m.ensureDocker(ctx, onLog); err != nil {
		return err
	}
	logLine(onLog, "[QX] Pulling "+spec.Image)
	if err := m.run(ctx, "docker", "pull", spec.Image); err != nil {
		return err
	}
	_ = m.run(ctx, "docker", "rm", "-f", ContainerName)
	publish := spec.BindAddr + ":" + strconv.Itoa(spec.Port) + ":3306"
	args := []string{
		"run", "-d",
		"--name", ContainerName,
		"--restart", "unless-stopped",
		"-e", "MYSQL_ROOT_PASSWORD=" + password,
		"-e", "MARIADB_ROOT_PASSWORD=" + password,
		"-p", publish,
		"-v", m.dataDir() + ":/var/lib/mysql",
		spec.Image,
	}
	logLine(onLog, "[QX] Starting container "+ContainerName)
	if err := m.run(ctx, "docker", args...); err != nil {
		return err
	}
	return m.waitReady(ctx, 2*time.Minute)
}

func (m *Manager) installNative(ctx context.Context, spec InstallResult, password string, onLog func(string)) error {
	logLine(onLog, "[QX] Installing native packages")
	if err := m.runApt(ctx, "update"); err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}
	switch spec.Engine {
	case mysqlutil.EngineMariaDB:
		if err := m.runApt(ctx, "install", "-y", "mariadb-server", "mariadb-client"); err != nil {
			return err
		}
	case mysqlutil.EnginePercona:
		if err := m.installPerconaRepo(ctx, spec.Version, password, onLog); err != nil {
			return err
		}
	default:
		return ErrUnsupported
	}
	if err := m.writeNativeCNF(spec); err != nil {
		return err
	}
	unit := m.nativeUnit(spec.Engine)
	_ = m.run(ctx, "systemctl", "daemon-reload")
	if err := m.run(ctx, "systemctl", "enable", "--now", unit); err != nil {
		if err := m.run(ctx, "systemctl", "enable", "--now", systemdMySQL); err != nil {
			return err
		}
	}
	_ = m.bootstrapNativeRoot(ctx, password)
	return m.waitReady(ctx, 2*time.Minute)
}

func perconaNativeRepos(version string) []string {
	if version == mysqlutil.Version57 {
		return []string{"ps-57"}
	}
	// 8.0 community packages ended June 2026. ps-80 indexes are empty on
	// Ubuntu 26.04 and Debian 13; 8.4 LTS still ships percona-server-server.
	return []string{"ps-84-lts", "ps80"}
}

func perconaNativePackage(version string) string {
	if version == mysqlutil.Version57 {
		return "percona-server-server-5.7"
	}
	return "percona-server-server"
}

func (m *Manager) installPerconaRepo(ctx context.Context, version, password string, onLog func(string)) error {
	if err := m.runApt(ctx, "install", "-y", "wget", "gnupg2", "lsb-release", "curl", "ca-certificates"); err != nil {
		_ = m.runApt(ctx, "install", "-y", "wget", "gnupg", "lsb-release", "curl")
	}
	deb := filepath.Join(m.RootDir, "percona-release.deb")
	if err := m.run(ctx, "wget", "-O", deb, "https://repo.percona.com/apt/percona-release_latest.generic_all.deb"); err != nil {
		if err := m.run(ctx, "curl", "-fsSL", "-o", deb, "https://repo.percona.com/apt/percona-release_latest.generic_all.deb"); err != nil {
			return err
		}
	}
	if err := m.runApt(ctx, "install", "-y", deb); err != nil {
		if err := m.run(ctx, "dpkg", "-i", deb); err != nil {
			_ = m.runApt(ctx, "install", "-y", "-f")
		}
	}
	pkg := perconaNativePackage(version)
	_ = m.preseedPerconaRoot(ctx, password)
	var last error
	for _, repo := range perconaNativeRepos(version) {
		logLine(onLog, "[QX] Enabling Percona "+repo)
		if err := m.enablePerconaRepo(ctx, repo); err != nil {
			last = err
			continue
		}
		if err := m.runApt(ctx, "install", "-y", pkg); err != nil {
			last = err
			continue
		}
		return nil
	}
	if last == nil {
		last = fmt.Errorf("package %s has no installation candidate", pkg)
	}
	return fmt.Errorf("%w; this distro may not publish Percona %s — use Docker instead", last, version)
}

func (m *Manager) enablePerconaRepo(ctx context.Context, repo string) error {
	if err := m.run(ctx, "percona-release", "setup", "-y", repo); err != nil {
		if err := m.run(ctx, "percona-release", "enable", repo, "release"); err != nil {
			return err
		}
	} else {
		_ = m.run(ctx, "percona-release", "enable", repo, "release")
	}
	_ = m.run(ctx, "percona-release", "disable", "tools")
	return m.runApt(ctx, "update")
}

func (m *Manager) preseedPerconaRoot(ctx context.Context, password string) error {
	body := "percona-server-server percona-server-server/root-pass password " + password + "\n" +
		"percona-server-server percona-server-server/re-root-pass password " + password + "\n"
	path := filepath.Join(m.RootDir, "percona.debconf")
	if err := writeFileFn(path, []byte(body), 0o600); err != nil {
		return err
	}
	return m.run(ctx, "debconf-set-selections", path)
}

func (m *Manager) runApt(ctx context.Context, args ...string) error {
	return m.runEnv(ctx, []string{"DEBIAN_FRONTEND=noninteractive", "NEEDRESTART_MODE=a"}, "apt-get", args...)
}

func (m *Manager) writeNativeCNF(spec InstallResult) error {
	body := fmt.Sprintf("[mysqld]\nbind-address = %s\nport = %d\n", spec.BindAddr, spec.Port)
	candidates := []string{
		"/etc/mysql/mariadb.conf.d/99-qx.cnf",
		"/etc/mysql/mysql.conf.d/99-qx.cnf",
		"/etc/mysql/conf.d/99-qx.cnf",
		filepath.Join(m.RootDir, "99-qx.cnf"),
	}
	var last error
	for _, path := range candidates {
		if err := mkdirAllFn(filepath.Dir(path), 0o755); err != nil {
			last = err
			continue
		}
		if err := writeFileFn(path, []byte(body), 0o644); err != nil {
			last = err
			continue
		}
		return nil
	}
	if last == nil {
		last = errors.New("unable to write mysql config")
	}
	return last
}

func (m *Manager) bootstrapNativeRoot(ctx context.Context, password string) error {
	quoted := mysqlutil.QuoteString(password)
	return m.execNativeBootstrap(ctx, "ALTER USER 'root'@'localhost' IDENTIFIED BY "+quoted+"; FLUSH PRIVILEGES")
}

func (m *Manager) execNativeBootstrap(ctx context.Context, stmt string) error {
	cmd := commandContext(ctx, "mysql", "-u", "root", "-e", stmt)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err.Error(), msg)
	}
	return nil
}

func (m *Manager) ensureDocker(ctx context.Context, onLog func(string)) error {
	if m.run(ctx, "docker", "info") == nil {
		return nil
	}
	logLine(onLog, "[QX] Installing Docker")
	if err := m.runApt(ctx, "update"); err != nil {
		return err
	}
	if err := m.runApt(ctx, "install", "-y", "docker.io"); err != nil {
		return err
	}
	_ = m.run(ctx, "systemctl", "enable", "--now", "docker")
	return m.run(ctx, "docker", "info")
}

func (m *Manager) execSQL(ctx context.Context, statements ...string) error {
	if m.DryRun {
		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				return errors.New("empty sql")
			}
		}
		return nil
	}
	if !m.Status(ctx).Installed {
		return ErrNotInstalled
	}
	if !m.ping(ctx) {
		return ErrNotRunning
	}
	password, err := m.readPassword()
	if err != nil {
		return err
	}
	cfg, _ := m.readInstance()
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := m.execOne(ctx, cfg, password, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) execOne(ctx context.Context, cfg instanceConfig, password, stmt string) error {
	var cmd *exec.Cmd
	if cfg.Method == mysqlutil.MethodDocker {
		cmd = commandContext(ctx, "docker", "exec", "-e", "MYSQL_PWD="+password, ContainerName, "mysql", "-uroot", "-e", stmt)
	} else {
		port := strconv.Itoa(cfg.Port)
		if port == "0" {
			port = strconv.Itoa(mysqlutil.DefaultPort)
		}
		cmd = commandContext(ctx, "mysql", "-uroot", "-h", "127.0.0.1", "-P", port, "-e", stmt)
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err.Error(), msg)
	}
	return nil
}

func (m *Manager) ping(ctx context.Context) bool {
	cfg, ok := m.readInstance()
	if !ok {
		return false
	}
	password, err := m.readPassword()
	if err != nil {
		return false
	}
	var cmd *exec.Cmd
	if cfg.Method == mysqlutil.MethodDocker {
		cmd = commandContext(ctx, "docker", "exec", "-e", "MYSQL_PWD="+password, ContainerName, "mysqladmin", "ping", "-uroot", "--silent")
	} else {
		port := strconv.Itoa(cfg.Port)
		if port == "0" {
			port = strconv.Itoa(mysqlutil.DefaultPort)
		}
		cmd = commandContext(ctx, "mysqladmin", "ping", "-uroot", "-h", "127.0.0.1", "-P", port, "--silent")
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
	}
	return cmd.Run() == nil
}

func (m *Manager) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if m.ping(ctx) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("mysql did not become ready")
}

func (m *Manager) nativeUnit(engine string) string {
	if engine == mysqlutil.EngineMariaDB {
		return systemdMaria
	}
	return systemdMySQL
}

func (m *Manager) dataDir() string {
	return filepath.Join(m.RootDir, "data")
}

func (m *Manager) instancePath() string {
	return filepath.Join(m.RootDir, "instance.json")
}

func (m *Manager) passwordPath() string {
	return filepath.Join(m.RootDir, "root.password")
}

func (m *Manager) writeInstance(cfg instanceConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return writeFileFn(m.instancePath(), append(raw, '\n'), 0o644)
}

func (m *Manager) readInstance() (instanceConfig, bool) {
	raw, err := readFileFn(m.instancePath())
	if err != nil {
		return instanceConfig{}, false
	}
	var cfg instanceConfig
	if json.Unmarshal(raw, &cfg) != nil {
		return instanceConfig{}, false
	}
	return cfg, true
}

func (m *Manager) writePassword(password string) error {
	return writeFileFn(m.passwordPath(), []byte(password+"\n"), 0o600)
}

func (m *Manager) readPassword() (string, error) {
	raw, err := readFileFn(m.passwordPath())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func (m *Manager) run(ctx context.Context, name string, args ...string) error {
	return m.runEnv(ctx, nil, name, args...)
}

func (m *Manager) runEnv(ctx context.Context, env []string, name string, args ...string) error {
	cmd := commandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err.Error(), msg)
	}
	return nil
}

func logLine(onLog func(string), line string) {
	if onLog != nil {
		onLog(line)
	}
}
