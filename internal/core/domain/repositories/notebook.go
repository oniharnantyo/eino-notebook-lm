package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// NotebookRepository defines the interface for notebook persistence
// This is part of the Domain layer and defines what the Application layer needs
type NotebookRepository interface {
	// Save saves a notebook (create or update)
	Save(ctx context.Context, notebook *entities.Notebook) error

	// FindByID finds a notebook by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error)

	// FindAll finds all notebooks with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*entities.Notebook, error)

	// FindByUserID finds notebooks by user ID with pagination
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Notebook, error)

	// FindByStatus finds notebooks by status
	FindByStatus(ctx context.Context, status string, limit, offset int) ([]*entities.Notebook, error)

	// FindByTags finds notebooks by tags
	FindByTags(ctx context.Context, tags []string, limit, offset int) ([]*entities.Notebook, error)

	// Search searches notebooks by title or content
	Search(ctx context.Context, query string, limit, offset int) ([]*entities.Notebook, error)

	// Delete deletes a notebook by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// Exists checks if a notebook exists
	Exists(ctx context.Context, id uuid.UUID) (bool, error)

	// Count returns the total count of notebooks
	Count(ctx context.Context) (int64, error)

	// CountByUserID returns the total count of notebooks for a user
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	FindByEmail(ctx context.Context, email string) (*entities.User, error)
	Save(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// CacheRepository defines the interface for caching
type CacheRepository interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl int) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
