//go:build !windows

package notify

func platformShow(title, message string) {}
