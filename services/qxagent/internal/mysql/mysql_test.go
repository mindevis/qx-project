package mysql

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootFromServerRoot(t *testing.T) {
	if got := RootFromServerRoot("/opt/qxsystem/server"); got != filepath.Join("/opt/qxsystem", "mysql") {
		t.Fatalf("got %q", got)
	}
	if got := RootFromServerRoot(""); got != DefaultRoot {
		t.Fatalf("empty: %q", got)
	}
}

func TestInstallDryRunAndSQL(t *testing.T) {
	m := NewManager(t.TempDir(), true)
	_, err := m.Install(context.Background(), InstallSpec{
		Engine: "mysql", Version: "8.0", Method: "docker", RootPassword: "secret",
	}, nil)
	if err == nil {
		t.Fatal("expected unsupported engine")
	}

	res, err := m.Install(context.Background(), InstallSpec{
		Engine:       "mariadb",
		Version:      "5.7",
		Method:       "docker",
		RootPassword: "s3cret",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Image != "mariadb:10.11" || res.Port != 3306 || res.BindAddr != "127.0.0.1" {
		t.Fatalf("%+v", res)
	}
	st := m.Status(context.Background())
	if !st.Installed || st.Running {
		t.Fatalf("status after install: %+v", st)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	st = m.Status(context.Background())
	if !st.Running {
		t.Fatal("expected running")
	}
	if err := m.CreateDatabase(context.Background(), "survival"); err != nil {
		t.Fatal(err)
	}
	if err := m.CreateDatabase(context.Background(), "bad-name"); err == nil {
		t.Fatal("expected invalid database")
	}
	if err := m.CreateUser(context.Background(), "plugin", "%", "pw"); err != nil {
		t.Fatal(err)
	}
	if err := m.Grant(context.Background(), "plugin", "%", "survival", []string{"SELECT", "INSERT"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Grant(context.Background(), "plugin", "%", "survival", []string{"SUPER"}); err == nil {
		t.Fatal("expected invalid privilege")
	}
	if err := m.DropUser(context.Background(), "plugin", "%"); err != nil {
		t.Fatal(err)
	}
	if err := m.DropDatabase(context.Background(), "survival"); err != nil {
		t.Fatal(err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if m.Status(context.Background()).Running {
		t.Fatal("expected stopped")
	}
}

func TestInstallNativeDryRun(t *testing.T) {
	m := NewManager(t.TempDir(), true)
	res, err := m.Install(context.Background(), InstallSpec{
		Engine:       "percona",
		Version:      "8",
		Method:       "native",
		BindAddr:     "0.0.0.0",
		Port:         3307,
		RootPassword: "rootpw",
	}, func(line string) {
		if !strings.Contains(line, "percona") && !strings.Contains(line, "dry-run") && !strings.Contains(line, "Installing") {
			t.Fatalf("log: %s", line)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Method != "native" || res.Image != "percona:8.0" || res.Port != 3307 {
		t.Fatalf("%+v", res)
	}
}
