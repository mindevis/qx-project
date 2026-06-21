//go:build windows

package device

import (
	"golang.org/x/sys/windows/registry"
)

func platformMachineRaw() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	val, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return ""
	}
	return val
}
