package minecraft

import (
	"path/filepath"
	"testing"

	"github.com/qxproject/qx/pkg/safepath"
)

func TestInstancePaths(t *testing.T) {
	root := t.TempDir()
	const (
		id  = "inst-abc"
		ver = "1.21.1"
	)
	dl := NewDownloader(root)

	must := func(path string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return path
	}

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"root", must(dl.InstanceRoot(id)), must(safepath.Join(root, "instances", id))},
		{"game", must(dl.InstanceGameDir(id)), must(safepath.Join(root, "instances", id))},
		{"assets", must(dl.InstanceAssetsDir(id)), must(safepath.Join(root, "instances", id, "assets"))},
		{"libraries", must(dl.InstanceLibrariesDir(id)), must(safepath.Join(root, "instances", id, "libraries"))},
		{"cache", must(dl.InstanceCacheDir(id)), must(safepath.Join(root, "instances", id, "cache"))},
		{"versions", must(dl.InstanceVersionsDir(id, ver)), must(safepath.Join(root, "instances", id, "versions", ver))},
		{
			"loader jar",
			must(dl.loaderClientJarPath(id, "libraries/net/minecraftforge/forge/forge.jar")),
			must(safepath.JoinRel(must(dl.InstanceRoot(id)), "libraries/net/minecraftforge/forge/forge.jar")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestInstanceRootRejectsTraversal(t *testing.T) {
	dl := NewDownloader(t.TempDir())
	if _, err := dl.InstanceRoot(".."); err == nil {
		t.Fatal("expected traversal error")
	}
	if _, err := dl.loaderClientJarPath("inst", "../escape.jar"); err == nil {
		t.Fatal("expected relative traversal error")
	}
}

func TestJavaStaysAtLauncherRoot(t *testing.T) {
	dl := NewDownloader("/data")
	javaRoot := dl.javaManager().RootDir
	want := filepath.Join("/data", "java")
	if javaRoot != want {
		t.Fatalf("java root = %q, want %q", javaRoot, want)
	}
}
