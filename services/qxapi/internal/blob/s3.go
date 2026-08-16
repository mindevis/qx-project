package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config is the MinIO / S3 connection used for connect-copy jars.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// S3 stores objects in MinIO (or any S3-compatible endpoint).
type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	endpoint, secure := normalizeEndpoint(cfg.Endpoint, cfg.UseSSL)
	if endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("minio endpoint, access key, and secret are required")
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		bucket = "qx"
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ok, err := client.BucketExists(checkCtx, bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !ok {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio create bucket: %w", err)
		}
	}
	return &S3{client: client, bucket: bucket}, nil
}

// Open connects to MinIO. Endpoint "memory" or empty is an in-process store for tests.
func Open(ctx context.Context, cfg Config) (Store, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" || strings.EqualFold(endpoint, "memory") {
		return NewMemory(), nil
	}
	var last error
	for attempt := 1; attempt <= 8; attempt++ {
		store, err := NewS3(ctx, cfg)
		if err == nil {
			return store, nil
		}
		last = err
		delay := time.Duration(attempt) * time.Second
		select {
		case <-ctx.Done():
			if last != nil {
				return nil, last
			}
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil, last
}

func (s *S3) Put(ctx context.Context, key string, data []byte) error {
	clean, err := sanitizeKey(key)
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, s.bucket, clean, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	clean, err := sanitizeKey(key)
	if err != nil {
		return nil, 0, err
	}
	obj, err := s.client.GetObject(ctx, s.bucket, clean, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, 0, err
	}
	return obj, info.Size, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	clean, err := sanitizeKey(key)
	if err != nil {
		return err
	}
	return s.client.RemoveObject(ctx, s.bucket, clean, minio.RemoveObjectOptions{})
}

func normalizeEndpoint(raw string, useSSL bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "https://"):
		return strings.TrimPrefix(raw, "https://"), true
	case strings.HasPrefix(raw, "http://"):
		return strings.TrimPrefix(raw, "http://"), false
	default:
		return raw, useSSL
	}
}
