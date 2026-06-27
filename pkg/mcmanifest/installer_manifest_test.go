package mcmanifest

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchVersionMetaFromInstaller(t *testing.T) {
	versionJSON := []byte(`{
		"id": "1.20.1-forge-47.4.20",
		"inheritsFrom": "1.20.1",
		"mainClass": "cpw.mods.bootstraplauncher.BootstrapLauncher",
		"arguments": {
			"game": ["--launchTarget", "forgeclient"],
			"jvm": ["-Xmx2G"]
		},
		"libraries": [{"name":"net.minecraftforge:forge:1.20.1-47.4.20","url":"https://maven.minecraftforge.net/"}]
	}`)

	var installer bytes.Buffer
	zw := zip.NewWriter(&installer)
	w, err := zw.Create("version.json")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write(versionJSON); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(installer.Bytes())
	}))
	t.Cleanup(srv.Close)

	client := &Client{HTTPClient: srv.Client()}
	meta, err := client.fetchVersionMetaFromInstaller(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if meta.MainClass != "cpw.mods.bootstraplauncher.BootstrapLauncher" || meta.InheritsFrom != "1.20.1" {
		t.Fatalf("meta: %+v", meta)
	}
}

func TestClientArtifactFromInstaller(t *testing.T) {
	installProfile := []byte(`{
		"data": {
			"PATCHED": {"client": "[net.minecraftforge:forge:1.20.1-47.4.20:client]"},
			"PATCHED_SHA": {"client": "'d8e1d9169ab0f6f3ff0ad41eccb3496f202d5484'"}
		}
	}`)
	var installer bytes.Buffer
	zw := zip.NewWriter(&installer)
	w, _ := zw.Create("install_profile.json")
	_, _ = w.Write(installProfile)
	_ = zw.Close()

	artifact, err := clientArtifactFromInstaller(installer.Bytes())
	if err != nil {
		t.Fatalf("artifact: %v", err)
	}
	want := "libraries/net/minecraftforge/forge/1.20.1-47.4.20/forge-1.20.1-47.4.20-client.jar"
	if artifact.RelativePath != want {
		t.Fatalf("path: %s", artifact.RelativePath)
	}
	if artifact.Sha1 != "d8e1d9169ab0f6f3ff0ad41eccb3496f202d5484" {
		t.Fatalf("sha1: %s", artifact.Sha1)
	}
}

func TestDefaultLoaderClientJar(t *testing.T) {
	artifact := DefaultLoaderClientJar(LoaderForge, "1.20.1", "47.4.20")
	want := "libraries/net/minecraftforge/forge/1.20.1-47.4.20/forge-1.20.1-47.4.20-client.jar"
	if artifact.RelativePath != want {
		t.Fatalf("path: %s", artifact.RelativePath)
	}
}

func TestForgeInstallerURL(t *testing.T) {
	url := forgeInstallerURL("1.20.1", "47.4.20")
	want := "https://maven.minecraftforge.net/net/minecraftforge/forge/1.20.1-47.4.20/forge-1.20.1-47.4.20-installer.jar"
	if url != want {
		t.Fatalf("url: %s", url)
	}
}
