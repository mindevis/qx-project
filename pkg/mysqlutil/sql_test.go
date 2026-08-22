package mysqlutil

import "testing"

func TestQuoteAndValidate(t *testing.T) {
	if !ValidIdent("luckperms") || ValidIdent("bad-name") || ValidIdent("1start") {
		t.Fatal("ident")
	}
	if !ValidUsername("plugin", "8.0") || ValidUsername("thisnameiswaytoolong1", "5.7") {
		t.Fatal("username")
	}
	ident, err := QuoteIdent("luck_perms")
	if err != nil || ident != "`luck_perms`" {
		t.Fatalf("quote: %s %v", ident, err)
	}
	if _, err := QuoteIdent("db;drop"); err == nil {
		t.Fatal("expected invalid ident")
	}
}

func TestGrantSQL(t *testing.T) {
	sql, err := GrantSQL("plugin", "%", "survival", []string{"SELECT", "insert"})
	if err != nil {
		t.Fatal(err)
	}
	if sql != "GRANT SELECT, INSERT ON `survival`.* TO 'plugin'@'%'" {
		t.Fatalf("grant: %s", sql)
	}
	if _, err := GrantSQL("plugin", "%", "survival", []string{"FILE"}); err == nil {
		t.Fatal("expected invalid privilege")
	}
	stmts, err := ApplyGrantStatements("plugin", "localhost", "hub", []string{"SELECT"})
	if err != nil || len(stmts) != 3 {
		t.Fatalf("apply: %v %v", stmts, err)
	}
	revoke, err := ApplyGrantStatements("plugin", "localhost", "hub", nil)
	if err != nil || len(revoke) != 2 {
		t.Fatalf("revoke: %v %v", revoke, err)
	}
}

func TestEngineMapping(t *testing.T) {
	img, err := DockerImage("mariadb", "5.7")
	if err != nil || img != "mariadb:10.11" {
		t.Fatalf("mariadb 5.7: %s %v", img, err)
	}
	img, err = DockerImage("MariaDB", "8")
	if err != nil || img != "mariadb:11.4" {
		t.Fatalf("mariadb 8: %s %v", img, err)
	}
	img, err = DockerImage("percona", "8.0")
	if err != nil || img != "percona:8.0" {
		t.Fatalf("percona 8: %s %v", img, err)
	}
	engine, version, err := NormalizeEngineVersion("PERCONA", "8")
	if err != nil || engine != "percona" || version != "8.0" {
		t.Fatalf("normalize: %s %s %v", engine, version, err)
	}
	if PackageVersion("mariadb", "5.7") != "10.11" {
		t.Fatal("package version")
	}
	if _, err := DockerImage("mysql", "8.0"); err == nil {
		t.Fatal("expected unsupported engine")
	}
}

func TestCreateUserSQL(t *testing.T) {
	sql, err := CreateUserSQL("core", "%", "p'ass")
	if err != nil || sql != "CREATE USER 'core'@'%' IDENTIFIED BY 'p''ass'" {
		t.Fatalf("%q %v", sql, err)
	}
	if _, err := CreateUserSQL("core", "%", ""); err == nil {
		t.Fatal("expected password required")
	}
}
