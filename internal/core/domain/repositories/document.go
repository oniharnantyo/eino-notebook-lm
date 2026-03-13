package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// DocumentRepository defines the interface for document persistence
// TODO: Implement repository methods based on requirements
type DocumentRepository interface {
	// Save saves a document (create or update)
	Save(ctx context.Context, document *entities.Document) error

	// FindByID finds a document by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Document, error)

	// FindAll finds all documents with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*entities.Document, error)

	// Delete deletes a document by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// Exists checks if a document exists
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// Count returns the total count of documents
	Count(ctx context.Context) (int64, error)
}
