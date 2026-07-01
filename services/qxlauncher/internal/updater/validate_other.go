//go:build !windows

package updater

func validateWindowsExecutable(string) error { return nil }
