package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ArtifactRepository defines the interface for artifact persistence operations
type ArtifactRepository interface {
	// Create creates a new artifact
	Create(ctx context.Context, artifact *entities.Artifact) error

	// GetByID retrieves an artifact by ID
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Artifact, error)

	// GetByNotebookID retrieves all artifacts for a notebook
	GetByNotebookID(ctx context.Context, notebookID uuid.UUID) ([]*entities.Artifact, error)

	// GetByNotebookIDAndType retrieves artifacts of a specific type for a notebook
	GetByNotebookIDAndType(ctx context.Context, notebookID uuid.UUID, artifactType entities.ArtifactType) ([]*entities.Artifact, error)

	// GetByNotebookIDAndStatus retrieves artifacts with a specific status for a notebook
	GetByNotebookIDAndStatus(ctx context.Context, notebookID uuid.UUID, status entities.ArtifactStatus) ([]*entities.Artifact, error)

	// Update updates an existing artifact
	Update(ctx context.Context, artifact *entities.Artifact) error

	// Delete soft-deletes an artifact
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves artifacts with pagination
	List(ctx context.Context, filter ArtifactFilter) ([]*entities.Artifact, int, error)
}

// ArtifactFilter defines filtering options for listing artifacts
type ArtifactFilter struct {
	NotebookID *uuid.UUID
	Type       *entities.ArtifactType
	Status     *entities.ArtifactStatus
	Limit      int
	Offset     int
	OrderBy    string // "created_at", "title", "updated_at", etc.
}
