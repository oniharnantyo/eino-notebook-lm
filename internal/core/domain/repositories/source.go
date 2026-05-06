package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SourceRepository defines the interface for source persistence operations
type SourceRepository interface {
	// Create creates a new source
	Create(ctx context.Context, source *entities.Source) error

	// GetByID retrieves a source by ID
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Source, error)

	// GetByNotebookID retrieves all sources for a notebook
	GetByNotebookID(ctx context.Context, notebookID uuid.UUID) ([]*entities.Source, error)

	// GetByURI retrieves a source by its URI (useful for deduplication)
	GetByURI(ctx context.Context, notebookID uuid.UUID, uri string) (*entities.Source, error)

	// Update updates an existing source
	Update(ctx context.Context, source *entities.Source) error

	// Delete soft-deletes a source
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves sources with pagination
	List(ctx context.Context, filter SourceFilter) ([]*entities.Source, int, error)

	// ListSourceSummariesByID retrieves source summaries by IDs
	ListSourceSummariesByID(ctx context.Context, ids []uuid.UUID) ([]*entities.Source, error)

	// IncrementChunkCount atomically increments the chunk counter
	IncrementChunkCount(ctx context.Context, id uuid.UUID) error
}

// SourceFilter defines filtering options for listing sources
type SourceFilter struct {
	NotebookID  *uuid.UUID
	ContentType *entities.ContentType
	Limit       int
	Offset      int
	OrderBy     string // "created_at", "title", "chunk_count", etc.
}
