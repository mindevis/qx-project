package api

import "testing"

func TestParseUserContentDownload(t *testing.T) {
	url, filename, err := parseUserContentDownload(
		"https://ci.lucko.me/job/LuckPerms/lastSuccessfulBuild/artifact/bukkit/loader/build/libs/LuckPerms-Bukkit-5.4.jar",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if filename != "LuckPerms-Bukkit-5.4.jar" {
		t.Fatalf("filename: %s", filename)
	}
	if url == "" {
		t.Fatal("expected url")
	}

	_, filename, err = parseUserContentDownload("https://example.com/download.php", "Vault.jar")
	if err != nil {
		t.Fatal(err)
	}
	if filename != "Vault.jar" {
		t.Fatalf("override filename: %s", filename)
	}

	if _, _, err := parseUserContentDownload("http://example.com/plugin.jar", ""); err == nil {
		t.Fatal("expected http rejected")
	}
	if _, _, err := parseUserContentDownload("https://127.0.0.1/plugin.jar", ""); err == nil {
		t.Fatal("expected ip rejected")
	}
	if _, _, err := parseUserContentDownload("https://localhost/plugin.jar", ""); err == nil {
		t.Fatal("expected localhost rejected")
	}
	if _, _, err := parseUserContentDownload("https://example.com/download.php", ""); err == nil {
		t.Fatal("expected missing filename rejected")
	}
	if _, _, err := parseUserContentDownload("https://example.com/secret.txt", ""); err == nil {
		t.Fatal("expected non-jar rejected")
	}
	if _, _, err := parseUserContentDownload("https://metadata.google.internal/plugin.jar", ""); err == nil {
		t.Fatal("expected .internal rejected")
	}
}
