package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Presigner is implemented by backends that can hand out time-limited direct download URLs.
type Presigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type S3Config struct {
	Endpoint  string // host[:port], without scheme (for AWS: s3.amazonaws.com or s3.<region>.amazonaws.com)
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Prefix    string // optional key prefix, e.g. "imageflow/"
}

// S3 stores objects in any S3-compatible service (AWS S3, Cloudflare R2, MinIO, Backblaze B2, ...).
// Unlike Local it is shared by every replica and survives container restarts.
type S3 struct {
	client *minio.Client
	bucket string
	prefix string
}

// NewS3 connects and checks that the bucket exists, so a bad configuration fails at start, not on the first image.
func NewS3(ctx context.Context, cfg S3Config) (*S3, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, errors.New("S3 endpoint and bucket are required")
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	ok, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", cfg.Bucket, err)
	}
	if !ok {
		return nil, fmt.Errorf("bucket %q does not exist", cfg.Bucket)
	}
	prefix := strings.TrimLeft(cfg.Prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &S3{client: client, bucket: cfg.Bucket, prefix: prefix}, nil
}

func (s *S3) objectName(key string) (string, error) {
	if key == "" || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.ContainsRune(key, 0) {
		return "", errors.New("invalid storage key")
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." || part == "." || part == "" {
			return "", errors.New("invalid storage key")
		}
	}
	return s.prefix + key, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	name, err := s.objectName(key)
	if err != nil {
		return 0, err
	}
	info, err := s.client.PutObject(ctx, s.bucket, name, r, -1, minio.PutObjectOptions{
		ContentType: mime.TypeByExtension(path.Ext(key)),
	})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	name, err := s.objectName(key)
	if err != nil {
		return nil, ErrNotFound
	}
	obj, err := s.client.GetObject(ctx, s.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return nil, mapS3Error(err)
	}
	// GetObject is lazy: Stat surfaces a missing object here instead of on the first Read.
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, mapS3Error(err)
	}
	return obj, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	name, err := s.objectName(key)
	if err != nil {
		return nil // an invalid key cannot name an object
	}
	// S3 reports success for a missing object, which is what idempotent cleanup needs.
	return s.client.RemoveObject(ctx, s.bucket, name, minio.RemoveObjectOptions{})
}

func (s *S3) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	name, err := s.objectName(key)
	if err != nil {
		return "", ErrNotFound
	}
	// A missing object must not yield a URL that only fails later.
	if _, err := s.client.StatObject(ctx, s.bucket, name, minio.StatObjectOptions{}); err != nil {
		return "", mapS3Error(err)
	}
	u, err := s.client.PresignedGetObject(ctx, s.bucket, name, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func mapS3Error(err error) error {
	switch minio.ToErrorResponse(err).Code {
	case "NoSuchKey", "NoSuchObject", "NotFound":
		return ErrNotFound
	}
	return err
}
