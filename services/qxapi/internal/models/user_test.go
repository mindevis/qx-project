package models

import "testing"

func TestUserTableName(t *testing.T) {
	if (User{}).TableName() != "users" {
		t.Fatal("unexpected table name")
	}
}
