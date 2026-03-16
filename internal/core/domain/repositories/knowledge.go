package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// KnowledgeRepository defines the interface for knowledge persistence
type KnowledgeRepository interface {
	// Save saves a knowledge (create or update)
	Save(ctx context.Context, knowledge *entities.Knowledge) error

	// FindByID finds a knowledge by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Knowledge, error)

	// FindBySourceID retrieves all knowledge chunks for a source
	GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Knowledge, error)

	// FindAll finds all knowledges with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error)

	// Delete deletes a knowledge by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteBySourceID deletes all knowledges for a source
	DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error

	// Exists checks if a knowledge exists
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// Count returns the total count of knowledges
	Count(ctx context.Context) (int64, error)

	// CountBySourceID returns the number of knowledge chunks for a source
	CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error)
}
