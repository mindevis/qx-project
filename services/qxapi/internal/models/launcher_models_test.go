package models

import "testing"

func TestLauncherModelTableNames(t *testing.T) {
	if (GuestSession{}).TableName() != "guest_sessions" {
		t.Fatal("guest_sessions table")
	}
	if (LauncherDevice{}).TableName() != "launcher_devices" {
		t.Fatal("launcher_devices table")
	}
	if (LauncherInstance{}).TableName() != "launcher_instances" {
		t.Fatal("launcher_instances table")
	}
}
