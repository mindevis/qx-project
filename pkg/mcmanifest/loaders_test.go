package mcmanifest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildProfileManifestFromURL(t *testing.T) {
	profileJSON := []byte(`{
		"id": "fabric",
		"mainClass": "net.fabricmc.loader.impl.launch.knot.KnotClient",
		"assetIndex": {"id":"1.20.1","sha1":"abc","size":1,"totalSize":2,"url":"https://example/assets.json"},
		"downloads": {"client": {"sha1":"jar","size":100,"url":"https://example/client.jar"}},
		"libraries": [{"name":"net.fabricmc:fabric-loader:0.16.14","downloads":{"artifact":{"sha1":"lib","size":1,"url":"https://example/loader.jar"}}}],
		"javaVersion": {"component":"java-runtime-gamma","majorVersion":17},
		"arguments": {
			"game": ["--username", "${auth_player_name}", {"rules":[{"action":"allow"}],"value":["--gameDir","${game_directory}"]}],
			"jvm": ["-Xmx2G"]
		}
	}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(profileJSON)
	}))
	t.Cleanup(srv.Close)

	client := &Client{HTTPClient: srv.Client()}
	manifest, err := client.buildProfileManifest(
		context.Background(),
		"inst-fabric",
		"Fabric Test",
		"1.20.1",
		LoaderFabric,
		"0.16.14",
		srv.URL,
		"windows",
	)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if manifest.MainClass != "net.fabricmc.loader.impl.launch.knot.KnotClient" {
		t.Fatalf("main class: %s", manifest.MainClass)
	}
	if manifest.LoaderVersion != "0.16.14" || len(manifest.GameArguments) < 4 {
		t.Fatalf("args: %+v", manifest.GameArguments)
	}
	if len(manifest.JVMArguments) != 1 {
		t.Fatalf("jvm args: %+v", manifest.JVMArguments)
	}
}

func TestBuildInstanceManifestRequiresLoaderVersion(t *testing.T) {
	client := NewClient()
	_, err := client.BuildInstanceManifest(context.Background(), "i", "n", "1.20.1", LoaderForge, "", "")
	if err == nil {
		t.Fatal("expected loader version error")
	}
}

func TestUnsupportedLoader(t *testing.T) {
	client := NewClient()
	_, err := client.BuildInstanceManifest(context.Background(), "i", "n", "1.20.1", "paper", "1", "")
	if err == nil {
		t.Fatal("expected unsupported loader error")
	}
}

func TestMavenArtifactURL(t *testing.T) {
	url := mavenArtifactURL("https://maven.fabricmc.net/", "org.ow2.asm:asm:9.8")
	if url != "https://maven.fabricmc.net/org/ow2/asm/asm/9.8/asm-9.8.jar" {
		t.Fatalf("url: %s", url)
	}
}

func TestNormalizeFabricLibrary(t *testing.T) {
	lib := normalizeLibrary(Library{
		Name:    "org.ow2.asm:asm:9.8",
		RepoURL: "https://maven.fabricmc.net/",
		Sha1:    "abc",
	})
	if lib.Downloads == nil || lib.Downloads.Artifact == nil || lib.Downloads.Artifact.URL == "" {
		t.Fatalf("downloads: %+v", lib.Downloads)
	}
}

func TestFlattenArgumentList(t *testing.T) {
	raw := json.RawMessage(`{"rules":[{"action":"allow","os":{"name":"windows"}}],"value":["--foo","bar"]}`)
	args := flattenArgumentList([]json.RawMessage{json.RawMessage(`"--username"`), raw}, "windows")
	if len(args) != 3 || args[0] != "--username" || args[1] != "--foo" {
		t.Fatalf("args: %+v", args)
	}
}
