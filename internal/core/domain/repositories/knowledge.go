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

	// FindByNotebookID finds knowledges by notebook ID with pagination
	FindByNotebookID(ctx context.Context, notebookID uuid.UUID, limit, offset int) ([]*entities.Knowledge, error)

	// FindByNotebookIDAndSourceType finds knowledges by notebook ID and source type
	FindByNotebookIDAndSourceType(ctx context.Context, notebookID uuid.UUID, sourceType string, limit, offset int) ([]*entities.Knowledge, error)

	// FindAll finds all knowledges with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error)

	// Delete deletes a knowledge by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByNotebookID deletes all knowledges for a notebook
	DeleteByNotebookID(ctx context.Context, notebookID uuid.UUID) error

	// Exists checks if a knowledge exists
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// Count returns the total count of knowledges
	Count(ctx context.Context) (int64, error)

	// CountByNotebookID returns the total count of knowledges for a notebook
	CountByNotebookID(ctx context.Context, notebookID uuid.UUID) (int64, error)
}
