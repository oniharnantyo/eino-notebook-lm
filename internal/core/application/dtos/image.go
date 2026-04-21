package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ImageResponse represents an image response
type ImageResponse struct {
	ID          uuid.UUID      `json:"id"`
	SourceID    uuid.UUID      `json:"source_id"`
	S3Key       string         `json:"s3_key"`
	Format      string         `json:"format"`
	Width       int            `json:"width"`
	Height      int            `json:"height"`
	Description string         `json:"description"`
	PageNumber  int            `json:"page_number"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
