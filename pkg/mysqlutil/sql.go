package mysqlutil

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	EngineMariaDB = "mariadb"
	EnginePercona = "percona"
	MethodDocker  = "docker"
	MethodNative  = "native"
	Version57     = "5.7"
	Version80     = "8.0"
	DefaultBind   = "127.0.0.1"
	DefaultPort   = 3306
)

// PrivilegeCatalog is the whitelist shown in the panel and enforced on GRANT.
var PrivilegeCatalog = []string{
	"ALL",
	"SELECT",
	"INSERT",
	"UPDATE",
	"DELETE",
	"CREATE",
	"DROP",
	"ALTER",
	"INDEX",
	"REFERENCES",
	"CREATE TEMPORARY TABLES",
	"LOCK TABLES",
	"EXECUTE",
	"CREATE VIEW",
	"SHOW VIEW",
	"TRIGGER",
	"EVENT",
	"CREATE ROUTINE",
	"ALTER ROUTINE",
}

var allowedPrivileges = func() map[string]struct{} {
	out := make(map[string]struct{}, len(PrivilegeCatalog))
	for _, p := range PrivilegeCatalog {
		out[p] = struct{}{}
	}
	return out
}()

func ValidIdent(name string) bool {
	return validIdentLen(name, 64)
}

func ValidUsername(name, version string) bool {
	max := 32
	if NormalizeVersion(version) == Version57 {
		max = 16
	}
	return validIdentLen(name, max)
}

func validIdentLen(name string, max int) bool {
	if name == "" || len(name) > max {
		return false
	}
	for i, r := range name {
		if unicode.IsLetter(r) || r == '_' {
			continue
		}
		if unicode.IsDigit(r) && i > 0 {
			continue
		}
		return false
	}
	return true
}

func ValidHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || host == "%" || host == "localhost" || host == "127.0.0.1" {
		return true
	}
	if len(host) > 255 {
		return false
	}
	for _, r := range host {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' || r == '%' {
			continue
		}
		return false
	}
	return true
}

func QuoteIdent(name string) (string, error) {
	if !ValidIdent(name) {
		return "", fmt.Errorf("invalid mysql identifier")
	}
	return "`" + name + "`", nil
}

func QuoteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func NormalizePrivileges(privs []string) ([]string, error) {
	if len(privs) == 0 {
		return nil, fmt.Errorf("privileges required")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(privs))
	for _, raw := range privs {
		p := strings.ToUpper(strings.TrimSpace(raw))
		if p == "ALL PRIVILEGES" {
			p = "ALL"
		}
		if _, ok := allowedPrivileges[p]; !ok {
			return nil, fmt.Errorf("invalid privilege %q", raw)
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("privileges required")
	}
	return out, nil
}

func PrivilegeSQL(privs []string) string {
	parts := make([]string, 0, len(privs))
	for _, p := range privs {
		if p == "ALL" {
			parts = append(parts, "ALL PRIVILEGES")
			continue
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, ", ")
}

func CreateDatabaseSQL(name string) (string, error) {
	ident, err := QuoteIdent(name)
	if err != nil {
		return "", err
	}
	return "CREATE DATABASE IF NOT EXISTS " + ident + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", nil
}

func DropDatabaseSQL(name string) (string, error) {
	ident, err := QuoteIdent(name)
	if err != nil {
		return "", err
	}
	return "DROP DATABASE IF EXISTS " + ident, nil
}

func CreateUserSQL(user, host, password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("mysql password required")
	}
	account, err := userHostSQL(user, host, "")
	if err != nil {
		return "", err
	}
	return "CREATE USER " + account + " IDENTIFIED BY " + QuoteString(password), nil
}

func DropUserSQL(user, host string) (string, error) {
	account, err := userHostSQL(user, host, "")
	if err != nil {
		return "", err
	}
	return "DROP USER " + account, nil
}

func GrantSQL(user, host, database string, privs []string) (string, error) {
	account, err := userHostSQL(user, host, "")
	if err != nil {
		return "", err
	}
	db, err := QuoteIdent(database)
	if err != nil {
		return "", err
	}
	clean, err := NormalizePrivileges(privs)
	if err != nil {
		return "", err
	}
	return "GRANT " + PrivilegeSQL(clean) + " ON " + db + ".* TO " + account, nil
}

func RevokeAllSQL(user, host, database string) (string, error) {
	account, err := userHostSQL(user, host, "")
	if err != nil {
		return "", err
	}
	db, err := QuoteIdent(database)
	if err != nil {
		return "", err
	}
	return "REVOKE ALL PRIVILEGES ON " + db + ".* FROM " + account, nil
}

func ApplyGrantStatements(user, host, database string, privs []string) ([]string, error) {
	revoke, err := RevokeAllSQL(user, host, database)
	if err != nil {
		return nil, err
	}
	if len(privs) == 0 {
		return []string{revoke, "FLUSH PRIVILEGES"}, nil
	}
	grant, err := GrantSQL(user, host, database, privs)
	if err != nil {
		return nil, err
	}
	return []string{revoke, grant, "FLUSH PRIVILEGES"}, nil
}

func userHostSQL(user, host, version string) (string, error) {
	if version == "" {
		version = Version80
	}
	if !ValidUsername(user, version) {
		return "", fmt.Errorf("invalid mysql username")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "%"
	}
	if !ValidHost(host) {
		return "", fmt.Errorf("invalid mysql host")
	}
	return QuoteString(user) + "@" + QuoteString(host), nil
}

func DockerImage(engine, version string) (string, error) {
	engine, version, err := NormalizeEngineVersion(engine, version)
	if err != nil {
		return "", err
	}
	switch engine {
	case EngineMariaDB:
		if version == Version57 {
			return "mariadb:10.11", nil
		}
		return "mariadb:11.4", nil
	case EnginePercona:
		if version == Version57 {
			return "percona:5.7", nil
		}
		return "percona:8.0", nil
	}
	return "", fmt.Errorf("unsupported mysql engine/version")
}

func PackageVersion(engine, version string) string {
	engine, version, err := NormalizeEngineVersion(engine, version)
	if err != nil {
		return ""
	}
	if engine == EngineMariaDB {
		if version == Version57 {
			return "10.11"
		}
		return "11.4"
	}
	return version
}

func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "8" || version == "11.4" {
		return Version80
	}
	if version == "10.11" {
		return Version57
	}
	return version
}

func NormalizeEngineVersion(engine, version string) (string, string, error) {
	engine = strings.ToLower(strings.TrimSpace(engine))
	version = NormalizeVersion(version)
	switch engine {
	case EngineMariaDB, EnginePercona:
	default:
		return "", "", fmt.Errorf("unsupported mysql engine")
	}
	switch version {
	case Version57, Version80:
	default:
		return "", "", fmt.Errorf("unsupported mysql version")
	}
	return engine, version, nil
}

func NormalizeMethod(method string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case MethodDocker, MethodNative:
		return strings.ToLower(strings.TrimSpace(method)), nil
	default:
		return "", fmt.Errorf("method must be docker or native")
	}
}

func NormalizeBind(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" || addr == "localhost" {
		return DefaultBind, nil
	}
	switch addr {
	case "127.0.0.1", "0.0.0.0":
		return addr, nil
	default:
		return "", fmt.Errorf("bind address must be 127.0.0.1 or 0.0.0.0")
	}
}

func NormalizePort(port int) (int, error) {
	if port == 0 {
		return DefaultPort, nil
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid mysql port")
	}
	return port, nil
}
