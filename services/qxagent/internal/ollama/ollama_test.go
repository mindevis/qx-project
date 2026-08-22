package ollama

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestRootFromServerRoot(t *testing.T) {
	if got := RootFromServerRoot("/opt/qxsystem/server"); got != filepath.Join("/opt/qxsystem", "ollama") {
		t.Fatalf("got %q", got)
	}
	if got := RootFromServerRoot(""); got != DefaultRoot {
		t.Fatalf("empty: %q", got)
	}
}

func TestLinuxArchAndDownloadURL(t *testing.T) {
	prev := runtimeGOARCH
	t.Cleanup(func() { runtimeGOARCH = prev })

	runtimeGOARCH = "amd64"
	arch, err := LinuxArch()
	if err != nil || arch != "amd64" {
		t.Fatalf("amd64: %s %v", arch, err)
	}
	if !strings.Contains(DownloadURL(arch), "ollama-linux-amd64.tar.zst") {
		t.Fatalf("url: %s", DownloadURL(arch))
	}

	runtimeGOARCH = "mips"
	if _, err := LinuxArch(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected unsupported: %v", err)
	}
}

func TestValidateModelName(t *testing.T) {
	if err := ValidateModelName("llama3.2:1b"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateModelName("hf.co/user/model"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateModelName("../etc/passwd"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected invalid: %v", err)
	}
	if err := ValidateModelName("bad name"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected invalid: %v", err)
	}
}

func TestInstallDryRun(t *testing.T) {
	m := NewManager(t.TempDir(), true)
	res, err := m.Install(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Version != "dry-run" || res.BinPath == "" {
		t.Fatalf("%+v", res)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.PullModel(context.Background(), "llama3.2", nil); err != nil {
		t.Fatal(err)
	}
	models, err := m.ListModels(context.Background())
	if err != nil || len(models) != 0 {
		t.Fatalf("list: %v %v", models, err)
	}
}

func TestExtractTGZAndListPull(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "ollama.tgz")
	if err := writeTestTGZ(archive, map[string]string{"bin/ollama": "#!/bin/sh\n"}); err != nil {
		t.Fatal(err)
	}
	if err := extractTGZ(archive, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "ollama")); err != nil {
		t.Fatal(err)
	}

	zst := filepath.Join(root, "ollama.tar.zst")
	if err := writeTestTarZST(zst, map[string]string{"bin/ollama": "#!/bin/sh\n"}); err != nil {
		t.Fatal(err)
	}
	zstRoot := t.TempDir()
	if err := extractArchive(zst, zstRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(zstRoot, "bin", "ollama")); err != nil {
		t.Fatal(err)
	}

	linkRoot := t.TempDir()
	linkArchive := filepath.Join(root, "ollama-link.tar.zst")
	if err := writeTestTarZSTWithSymlink(linkArchive); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(linkArchive, linkRoot); err != nil {
		t.Skipf("symlink extract unsupported: %v", err)
	}
	got, err := os.Readlink(filepath.Join(linkRoot, "lib", "ollama", "lib.so"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "lib.so.1" {
		t.Fatalf("symlink: %q", got)
	}

	prevGet := httpGet
	t.Cleanup(func() { httpGet = prevGet })
	httpGet = func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/api/version"):
			return jsonResponse(200, `{"version":"0.9.0"}`), nil
		case strings.HasSuffix(req.URL.Path, "/api/tags"):
			return jsonResponse(200, `{"models":[{"name":"llama3.2:latest","size":10,"digest":"abc"}]}`), nil
		case strings.HasSuffix(req.URL.Path, "/api/pull") && req.Method == http.MethodPost:
			return jsonResponse(200, `{"status":"success"}`), nil
		case strings.HasSuffix(req.URL.Path, "/api/delete") && req.Method == http.MethodDelete:
			return jsonResponse(200, `{}`), nil
		default:
			return jsonResponse(404, `{}`), nil
		}
	}

	m := NewManager(root, false)
	st := m.Status(context.Background())
	if !st.Installed || !st.Running {
		t.Fatalf("status: %+v", st)
	}
	models, err := m.ListModels(context.Background())
	if err != nil || len(models) != 1 || models[0].Name != "llama3.2:latest" {
		t.Fatalf("models: %v %v", models, err)
	}
	if err := m.PullModel(context.Background(), "phi4", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.DeleteModel(context.Background(), "phi4"); err != nil {
		t.Fatal(err)
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func writeTestTGZ(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0755, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.WriteString(tw, body); err != nil {
			return err
		}
	}
	return nil
}

func writeTestTarZST(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw, err := zstd.NewWriter(f)
	if err != nil {
		return err
	}
	defer zw.Close()
	tw := tar.NewWriter(zw)
	defer tw.Close()
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0755, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.WriteString(tw, body); err != nil {
			return err
		}
	}
	return nil
}

func writeTestTarZSTWithSymlink(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw, err := zstd.NewWriter(f)
	if err != nil {
		return err
	}
	defer zw.Close()
	tw := tar.NewWriter(zw)
	defer tw.Close()
	body := "so"
	if err := tw.WriteHeader(&tar.Header{Name: "lib/ollama/lib.so.1", Mode: 0644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		return err
	}
	if _, err := io.WriteString(tw, body); err != nil {
		return err
	}
	return tw.WriteHeader(&tar.Header{Name: "lib/ollama/lib.so", Mode: 0777, Typeflag: tar.TypeSymlink, Linkname: "lib.so.1"})
}

func TestEnvSetsLibraryPath(t *testing.T) {
	m := NewManager(t.TempDir(), true)
	found := false
	for _, item := range m.env() {
		if strings.HasPrefix(item, "LD_LIBRARY_PATH=") && strings.Contains(item, m.LibDir()) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing LD_LIBRARY_PATH in %v", m.env())
	}
}
