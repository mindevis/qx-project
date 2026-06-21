//go:build !windows && !linux && !darwin

package device

func platformMachineRaw() string {
	return ""
}
