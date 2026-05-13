package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Knowledge represents a chunk of information from a source.
type Knowledge struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	SourceID       uuid.UUID      `json:"source_id" db:"source_id"`
	Content        string         `json:"content" db:"content"`
	ChunkIndex     int            `json:"chunk_index" db:"chunk_index"`
	HeadingContext map[string]any `json:"heading_context" db:"heading_context"`
	FirstPage      int            `json:"first_page" db:"first_page"`
	LastPage       int            `json:"last_page" db:"last_page"`
	Metadata       map[string]any `json:"metadata" db:"metadata"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// NewKnowledge creates a new knowledge entity (chunk).
func NewKnowledge(sourceID uuid.UUID, content string, metadata map[string]any) (*Knowledge, error) {
	return &Knowledge{
		ID:        uuid.New(),
		SourceID:  sourceID,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now(),
	}, nil
}
