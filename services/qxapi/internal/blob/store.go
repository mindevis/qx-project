package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Store holds connect-copy jars outside MySQL.
type Store interface {
	Put(ctx context.Context, key string, data []byte) error
	Open(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
}

func sanitizeKey(key string) (string, error) {
	key = strings.TrimSpace(strings.ReplaceAll(key, `\`, "/"))
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("invalid blob key")
	}
	return key, nil
}

// Dir is a directory-backed store.
type Dir struct {
	root string
}

func NewDir(root string) (*Dir, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("blob dir is empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Dir{root: root}, nil
}

func (d *Dir) path(key string) (string, error) {
	clean, err := sanitizeKey(key)
	if err != nil {
		return "", err
	}
	return filepath.Join(d.root, filepath.FromSlash(clean)), nil
}

func (d *Dir) Put(_ context.Context, key string, data []byte) error {
	abs, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp := abs + ".part"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, abs)
}

func (d *Dir) Open(_ context.Context, key string) (io.ReadCloser, int64, error) {
	abs, err := d.path(key)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, err
	}
	return f, info.Size(), nil
}

func (d *Dir) Delete(_ context.Context, key string) error {
	abs, err := d.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Memory is an in-process store for tests.
type Memory struct {
	mu   sync.Mutex
	data map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{data: map[string][]byte{}}
}

func (m *Memory) Put(_ context.Context, key string, data []byte) error {
	clean, err := sanitizeKey(key)
	if err != nil {
		return err
	}
	cp := append([]byte(nil), data...)
	m.mu.Lock()
	m.data[clean] = cp
	m.mu.Unlock()
	return nil
}

func (m *Memory) Open(_ context.Context, key string) (io.ReadCloser, int64, error) {
	clean, err := sanitizeKey(key)
	if err != nil {
		return nil, 0, err
	}
	m.mu.Lock()
	raw, ok := m.data[clean]
	m.mu.Unlock()
	if !ok {
		return nil, 0, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(raw)), int64(len(raw)), nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	clean, err := sanitizeKey(key)
	if err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.data, clean)
	m.mu.Unlock()
	return nil
}
