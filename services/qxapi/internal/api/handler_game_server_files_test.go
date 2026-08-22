package api

import "testing"

func TestValidateFileManagerPath(t *testing.T) {
	got, err := validateFileManagerPath("plugins/configs")
	if err != nil {
		t.Fatal(err)
	}
	if got != "plugins/configs" {
		t.Fatalf("path: %s", got)
	}
	if _, err := validateFileManagerPath("../etc"); err == nil {
		t.Fatal("expected traversal rejected")
	}
	if _, err := validateFileManagerPath(""); err == nil {
		t.Fatal("expected empty rejected")
	}
	got, err = validateFileManagerPath("/world/datapacks/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "world/datapacks" {
		t.Fatalf("trimmed path: %s", got)
	}
}

func TestSanitizeFileManagerName(t *testing.T) {
	got, err := sanitizeFileManagerName(`C:\uploads\LuckPerms.jar`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "LuckPerms.jar" {
		t.Fatalf("name: %s", got)
	}
	if _, err := sanitizeFileManagerName(".."); err == nil {
		t.Fatal("expected .. rejected")
	}
	if joinFileManagerPath("plugins", "foo.jar") != "plugins/foo.jar" {
		t.Fatal("join nested")
	}
	if joinFileManagerPath("", "eula.txt") != "eula.txt" {
		t.Fatal("join root")
	}
}
