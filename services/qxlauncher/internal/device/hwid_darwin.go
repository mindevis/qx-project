//go:build darwin

package device

import (
	"os/exec"
	"regexp"
	"strings"
)

var platformUUIDRe = regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([^"]+)"`)

func platformMachineRaw() string {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}
	m := platformUUIDRe.FindSubmatch(out)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}
