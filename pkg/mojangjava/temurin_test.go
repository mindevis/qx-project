package mojangjava

import "testing"

func TestArchiveKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path, contentType, urlPath, want string
	}{
		{"temurin-25-download", "application/gzip", "/OpenJDK25U-jdk_x64_linux_hotspot_25.tgz", "tgz"},
		{"temurin-25-download", "application/octet-stream", "/OpenJDK25U-jdk_x64_linux_hotspot_25.0.2_linux_hotspot_25_36.tar.gz", "tgz"},
		{"temurin-25-download", "application/zip", "/OpenJDK25U-jdk_x64_windows_hotspot_25_36.zip", "zip"},
		{"temurin-25-download.zip", "application/octet-stream", "/jdk.zip", "zip"},
	}
	for _, tc := range cases {
		if got := archiveKind(tc.path, tc.contentType, tc.urlPath); got != tc.want {
			t.Fatalf("%q %q %q: got %q want %q", tc.contentType, tc.urlPath, tc.path, got, tc.want)
		}
	}
}
