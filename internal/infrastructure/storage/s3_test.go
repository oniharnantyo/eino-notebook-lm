package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NewS3StorageForTest creates a test S3Storage instance with a mock client.
// This is intended for testing purposes only.
func NewS3StorageForTest(bucket string) *S3Storage {
	// Create a minio client with test credentials
	// In a real test environment, you would use a test MinIO server
	client, _ := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("test-access", "test-secret", ""),
		Secure: false,
	})

	return &S3Storage{
		client: client,
		bucket: bucket,
	}
}

// NewMockS3Storage creates a mock S3Storage for testing that doesn't require a real server.
type MockS3Storage struct {
	Uploads   map[string][]byte
	Deletions []string
}

// NewMockS3Storage creates a new mock S3 storage.
func NewMockS3Storage() *MockS3Storage {
	return &MockS3Storage{
		Uploads:   make(map[string][]byte),
		Deletions: []string{},
	}
}

// Upload simulates uploading data to S3.
func (m *MockS3Storage) Upload(ctx context.Context, key string, data []byte) error {
	m.Uploads[key] = data
	return nil
}

// Delete simulates deleting data from S3.
func (m *MockS3Storage) Delete(ctx context.Context, key string) error {
	m.Deletions = append(m.Deletions, key)
	delete(m.Uploads, key)
	return nil
}

// EnsureBucket is a no-op for the mock.
func (m *MockS3Storage) EnsureBucket(ctx context.Context) error {
	return nil
}
