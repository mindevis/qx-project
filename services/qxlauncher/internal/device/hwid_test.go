package device

import (
	"testing"

	"github.com/google/uuid"
)

func TestMachineDeviceIDStableFromRaw(t *testing.T) {
	const raw = "test-machine-guid-12345"
	want := uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw)).String()
	got := uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw)).String()
	if got != want {
		t.Fatalf("uuid: got %q want %q", got, want)
	}
	if _, err := uuid.Parse(MachineDeviceID()); MachineDeviceID() != "" && err != nil {
		t.Fatalf("machine id not uuid: %q err=%v", MachineDeviceID(), err)
	}
}

func TestResolveDeviceIDUsesSavedFile(t *testing.T) {
	dir := t.TempDir()
	if err := SaveDeviceID(dir, "saved-dev"); err != nil {
		t.Fatal(err)
	}
	if got := ResolveDeviceID(dir); got != "saved-dev" {
		t.Fatalf("resolve: got %q", got)
	}
}
