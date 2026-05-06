package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/oniharnantyo/eino-notebook/internal/infrastructure/config"
)

// S3Storage handles file operations on S3 compatible storage (like MinIO)
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage creates a new S3 storage client
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	// Initialize minio client object.
	// For local MinIO, Secure is typically false.
	// In a production environment, this should be configurable.
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &S3Storage{
		client: minioClient,
		bucket: cfg.Bucket,
	}, nil
}

// Upload uploads data to the specified key in S3
func (s *S3Storage) Upload(ctx context.Context, key string, data []byte) error {
	reader := bytes.NewReader(data)
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

// Delete removes an object from S3
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// EnsureBucket checks if the bucket exists and creates it if it doesn't
func (s *S3Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}
