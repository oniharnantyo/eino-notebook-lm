package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ImageRepository defines the interface for image persistence and vector search
type ImageRepository interface {
	// Save saves an image
	Save(ctx context.Context, image *entities.Image) error

	// FindByID finds an image by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Image, error)

	// FindBySourceID retrieves all images for a source
	FindBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Image, error)

	// Delete deletes an image by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteBySourceID deletes all images for a source
	DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error

	// CountBySourceID returns the number of images for a source
	CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error)
}
