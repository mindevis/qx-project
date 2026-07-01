package minecraft

import (
	"path/filepath"
	"testing"
)

func TestInstancePaths(t *testing.T) {
	const (
		root = "/data"
		id   = "inst-abc"
		ver  = "1.21.1"
	)
	dl := NewDownloader(root)

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"root", dl.InstanceRoot(id), filepath.Join(root, "instances", id)},
		{"game", dl.InstanceGameDir(id), filepath.Join(root, "instances", id)},
		{"assets", dl.InstanceAssetsDir(id), filepath.Join(root, "instances", id, "assets")},
		{"libraries", dl.InstanceLibrariesDir(id), filepath.Join(root, "instances", id, "libraries")},
		{"cache", dl.InstanceCacheDir(id), filepath.Join(root, "instances", id, "cache")},
		{"versions", dl.InstanceVersionsDir(id, ver), filepath.Join(root, "instances", id, "versions", ver)},
		{
			"loader jar",
			dl.loaderClientJarPath(id, "libraries/net/minecraftforge/forge/forge.jar"),
			filepath.Join(root, "instances", id, "libraries", "net", "minecraftforge", "forge", "forge.jar"),
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

func TestJavaStaysAtLauncherRoot(t *testing.T) {
	dl := NewDownloader("/data")
	javaRoot := dl.javaManager().RootDir
	want := filepath.Join("/data", "java")
	if javaRoot != want {
		t.Fatalf("java root = %q, want %q", javaRoot, want)
	}
}
