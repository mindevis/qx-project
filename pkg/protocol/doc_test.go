package protocol

import "testing"

func TestPackageDoc(t *testing.T) {
	if testing.Short() {
		t.Skip("package marker")
	}
}
