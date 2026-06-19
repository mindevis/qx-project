package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qxproject/qx/pkg/mcmanifest"
)

func TestEnsureClientJar(t *testing.T) {
	dir := t.TempDir()
	body := []byte("fake-jar")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	manifest := &mcmanifest.InstanceLaunchManifest{
		InstanceID: "inst-1",
		MCVersion:  "1.21",
		MainClass:  "net.minecraft.client.main.Main",
		ClientJar:  mcmanifest.DownloadFile{URL: srv.URL, Sha1: ""},
	}
	dl := NewDownloader(dir)
	path, err := dl.EnsureClientJar(context.Background(), manifest)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("missing jar: %v", err)
	}
}

func TestBuildLaunchPlan(t *testing.T) {
	manifest := &mcmanifest.InstanceLaunchManifest{
		MCVersion: "1.21",
		MainClass: "net.minecraft.client.main.Main",
		AssetIndex: mcmanifest.AssetIndexRef{ID: "1.21"},
	}
	jar := filepath.Join(t.TempDir(), "1.21.jar")
	plan := BuildLaunchPlan(manifest, jar, []string{"/lib/a.jar"}, "/natives", "/assets", "/game", "Steve", "uuid-1", "")
	if plan.MainClass == "" || len(plan.Args) == 0 {
		t.Fatalf("plan: %+v", plan)
	}
	if plan.WorkingDir != "/game" {
		t.Fatalf("game dir: %s", plan.WorkingDir)
	}
}

func TestBuildClasspath(t *testing.T) {
	cp := BuildClasspath([]string{"a.jar", "b.jar"})
	if cp == "" || !strings.Contains(cp, "a.jar") {
		t.Fatalf("classpath: %q", cp)
	}
}

func TestMavenRelPath(t *testing.T) {
	p := MavenRelPath("com.mojang:patchy:1.3.9")
	if !strings.Contains(p, "patchy-1.3.9.jar") {
		t.Fatalf("path: %s", p)
	}
}
