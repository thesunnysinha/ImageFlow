// Package storage holds processed images. Local is the development implementation;
// an S3-compatible implementation can replace it behind the same interface.
package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrNotFound = errors.New("object not found")

type Storage interface {
	// Put stores r under key and returns the number of bytes written.
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &Local{root: abs}, nil
}

func (l *Local) path(key string) (string, error) {
	clean := filepath.Clean("/" + key) // anchored, so ".." cannot climb out
	if key == "" || strings.Contains(key, "\x00") || strings.HasSuffix(key, "/") || clean == "/" {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(l.root, clean), nil
}

// Put writes to a temporary file and renames it, so readers never see a partial object.
func (l *Local) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	p, err := l.path(key)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".put-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck
	n, err := io.Copy(tmp, &ctxReader{ctx: ctx, r: r})
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return 0, err
	}
	return n, os.Rename(tmp.Name(), p)
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := l.path(key)
	if err != nil {
		return nil, ErrNotFound
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c *ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}
