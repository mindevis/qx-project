package protocol

import "testing"

func TestEnabledDisabledContentFilenames(t *testing.T) {
	if got := EnabledContentFilename("sodium.jar.disabled"); got != "sodium.jar" {
		t.Fatalf("EnabledContentFilename: %q", got)
	}
	if got := EnabledContentFilename("sodium.jar"); got != "sodium.jar" {
		t.Fatalf("EnabledContentFilename passthrough: %q", got)
	}
	if got := DisabledContentFilename("sodium.jar"); got != "sodium.jar.disabled" {
		t.Fatalf("DisabledContentFilename: %q", got)
	}
	if got := DisabledContentFilename("sodium.jar.DISABLED"); got != "sodium.jar.disabled" {
		t.Fatalf("DisabledContentFilename from disabled: %q", got)
	}
	if IsContentDisabledFilename(".disabled") {
		t.Fatal("bare .disabled should not count as a content file")
	}
	if !IsContentDisabledFilename("pack.zip.disabled") {
		t.Fatal("expected pack.zip.disabled")
	}
	if !SameContentFilename("Sodium.JAR", "sodium.jar.disabled") {
		t.Fatal("expected SameContentFilename to ignore suffix and case")
	}
	if SameContentFilename("", ".disabled") {
		t.Fatal("empty stem should not match")
	}
}
