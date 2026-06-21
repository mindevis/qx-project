//go:build linux

package device

import (
	"os"
	"strings"
)

func platformMachineRaw() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if id := strings.TrimSpace(string(b)); id != "" {
			return id
		}
	}
	return ""
}
