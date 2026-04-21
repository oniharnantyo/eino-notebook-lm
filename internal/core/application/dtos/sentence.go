package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SentenceResponse represents a sentence response
type SentenceResponse struct {
	ID          uuid.UUID      `json:"id"`
	KnowledgeID uuid.UUID      `json:"knowledge_id"`
	Content     string         `json:"content"`
	Position    int            `json:"position"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
