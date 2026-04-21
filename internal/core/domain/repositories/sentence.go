package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SentenceRepository defines the interface for sentence persistence and vector search
type SentenceRepository interface {
	// Save saves a sentence
	Save(ctx context.Context, sentence *entities.Sentence) error

	// SaveBatch saves multiple sentences in a single operation
	SaveBatch(ctx context.Context, sentences []*entities.Sentence) error

	// FindByID finds a sentence by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Sentence, error)

	// FindByKnowledgeID retrieves all sentences for a knowledge chunk
	FindByKnowledgeID(ctx context.Context, knowledgeID uuid.UUID) ([]*entities.Sentence, error)

	// Delete deletes a sentence by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByKnowledgeID deletes all sentences for a knowledge chunk
	DeleteByKnowledgeID(ctx context.Context, knowledgeID uuid.UUID) error

	// DeleteBySourceID deletes all sentences associated with a source (via knowledge chunks)
	DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error
}
