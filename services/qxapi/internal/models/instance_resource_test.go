package models

import (
	"testing"
)

func TestInstanceResourceListValueScan(t *testing.T) {
	list := InstanceResourceList{
		{ProjectName: "Sodium", Filename: "sodium.jar", ResourceType: "mod"},
	}
	val, err := list.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	var decoded InstanceResourceList
	if err := decoded.Scan(val); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(decoded) != 1 || decoded[0].ProjectName != "Sodium" {
		t.Fatalf("unexpected decoded: %+v", decoded)
	}
}
