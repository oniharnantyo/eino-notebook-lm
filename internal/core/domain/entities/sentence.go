package entities

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Sentence represents a sentence within a knowledge chunk.
type Sentence struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	KnowledgeID uuid.UUID      `json:"knowledge_id" db:"knowledge_id"`
	Content     string         `json:"content" db:"content"`
	Embedding   []float32      `json:"embedding" db:"embedding"`
	Position    int            `json:"position" db:"position"`
	Metadata    map[string]any `json:"metadata" db:"metadata"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
}

// NewSentence creates a new sentence entity.
func NewSentence(knowledgeID uuid.UUID, content string, position int, metadata map[string]any) (*Sentence, error) {
	return &Sentence{
		ID:          uuid.New(),
		KnowledgeID: knowledgeID,
		Content:     content,
		Position:    position,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
	}, nil
}
