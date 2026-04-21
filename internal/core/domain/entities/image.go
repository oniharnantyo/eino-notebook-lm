package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Image represents an image extracted from a source.
type Image struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	SourceID    uuid.UUID      `json:"source_id" db:"source_id"`
	S3Key       string         `json:"s3_key" db:"s3_key"`
	Format      string         `json:"format" db:"format"`
	Width       int            `json:"width" db:"width"`
	Height      int            `json:"height" db:"height"`
	Description string         `json:"description" db:"description"`
	PageNumber  int            `json:"page_number" db:"page_number"`
	Embedding   []float32      `json:"embedding" db:"embedding"`
	Metadata    map[string]any `json:"metadata" db:"metadata"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

// NewImage creates a new image entity.
func NewImage(sourceID uuid.UUID, s3Key, format string, width, height int, description string, pageNumber int, metadata map[string]any) (*Image, error) {
	return &Image{
		ID:          uuid.New(),
		SourceID:    sourceID,
		S3Key:       s3Key,
		Format:      format,
		Width:       width,
		Height:      height,
		Description: description,
		PageNumber:  pageNumber,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}, nil
}
