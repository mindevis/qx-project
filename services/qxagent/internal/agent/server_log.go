package agent

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"time"
)

const serverLogPollInterval = 500 * time.Millisecond

func readRecentLogLines(path string, maxLines int) ([]string, error) {
	lines, _, err := readLogTail(path, 0)
	if err != nil {
		return nil, err
	}
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines, nil
}

func followServerLog(ctx context.Context, workDir string, onLine func(line string)) {
	if onLine == nil || workDir == "" {
		return
	}
	logPath := filepath.Join(workDir, "logs", "latest.log")
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(serverLogPollInterval):
		}
		lines, nextOffset, err := readLogTail(logPath, offset)
		if err != nil {
			continue
		}
		offset = nextOffset
		for _, line := range lines {
			if line != "" {
				onLine(line)
			}
		}
	}
}

func readLogTail(path string, offset int64) ([]string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, offset, err
	}
	if stat.Size() < offset {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, offset, err
	}
	nextOffset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return lines, offset, nil
	}
	return lines, nextOffset, nil
}
