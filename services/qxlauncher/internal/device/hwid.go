package device

import (
	"strings"

	"github.com/google/uuid"
)

// MachineDeviceID returns a stable UUID derived from platform-specific machine identity (HWID).
func MachineDeviceID() string {
	raw := strings.TrimSpace(platformMachineRaw())
	if raw == "" {
		return ""
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw)).String()
}
