package launcher

import (
	"context"
	"testing"
)

func TestUpdateInstanceMaxMemory(t *testing.T) {
	svc, _, _ := newLauncherService(t)
	ctx := context.Background()
	owner := Owner{UserID: "user-1"}

	inst, err := svc.CreateInstance(ctx, owner, CreateInstanceInput{
		Name:          "Test",
		MCVersion:     "1.21.1",
		Loader:        "fabric",
		LoaderVersion: "0.16.9",
	})
	if err != nil {
		t.Fatal(err)
	}

	ram := 4096
	updated, err := svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MaxMemoryMB: &ram,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.MaxMemoryMB == nil || *updated.MaxMemoryMB != 4096 {
		t.Fatalf("expected 4096 MB, got %+v", updated.MaxMemoryMB)
	}

	bad := 128
	_, err = svc.UpdateInstance(ctx, owner, inst.ID, UpdateInstanceInput{
		MaxMemoryMB: &bad,
	})
	if err != ErrValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}
