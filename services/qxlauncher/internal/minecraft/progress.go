package minecraft

import "fmt"

// OnProgress receives human-readable launch preparation updates (phase, message).
type ProgressFunc func(phase, message string)

func (d *Downloader) progress(phase, message string) {
	if d == nil || d.OnProgress == nil {
		return
	}
	d.OnProgress(phase, message)
}

func (d *Downloader) progressf(phase, format string, args ...any) {
	d.progress(phase, fmt.Sprintf(format, args...))
}
