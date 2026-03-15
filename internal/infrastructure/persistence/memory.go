package persistence

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// InMemoryNotebookRepository implements NotebookRepository in memory
// Useful for testing and development
type InMemoryNotebookRepository struct {
	mu      sync.RWMutex
	nbs     map[uuid.UUID]*entities.Notebook
}

// NewInMemoryNotebookRepository creates a new in-memory repository
func NewInMemoryNotebookRepository() repositories.NotebookRepository {
	return &InMemoryNotebookRepository{
		nbs: make(map[uuid.UUID]*entities.Notebook),
	}
}

// Save saves a notebook
func (r *InMemoryNotebookRepository) Save(ctx context.Context, notebook *entities.Notebook) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nbs[notebook.ID] = notebook
	return nil
}

// FindByID finds a notebook by ID
func (r *InMemoryNotebookRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	notebook, exists := r.nbs[id]
	if !exists {
		return nil, nil
	}
	return notebook, nil
}

// FindAll finds all notebooks with pagination
func (r *InMemoryNotebookRepository) FindAll(ctx context.Context, limit, offset int) ([]*entities.Notebook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entities.Notebook, 0, limit)
	count := 0
	skipped := 0

	for _, notebook := range r.nbs {
		if notebook.IsDeleted() {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, notebook)
		count++
	}

	return result, nil
}

// FindByUserID finds notebooks by user ID with pagination
func (r *InMemoryNotebookRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Notebook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entities.Notebook, 0, limit)
	count := 0
	skipped := 0

	for _, notebook := range r.nbs {
		if notebook.IsDeleted() || notebook.UserID != userID {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, notebook)
		count++
	}

	return result, nil
}

// FindByStatus finds notebooks by status
func (r *InMemoryNotebookRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*entities.Notebook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entities.Notebook, 0, limit)
	count := 0
	skipped := 0

	for _, notebook := range r.nbs {
		if notebook.Status.String() != status {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, notebook)
		count++
	}

	return result, nil
}

// FindByTags finds notebooks by tags
func (r *InMemoryNotebookRepository) FindByTags(ctx context.Context, tags []string, limit, offset int) ([]*entities.Notebook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entities.Notebook, 0, limit)
	count := 0
	skipped := 0

	for _, notebook := range r.nbs {
		if notebook.IsDeleted() {
			continue
		}
		if !containsTags(notebook.Tags, tags) {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, notebook)
		count++
	}

	return result, nil
}

// Search searches notebooks by title or content
func (r *InMemoryNotebookRepository) Search(ctx context.Context, query string, limit, offset int) ([]*entities.Notebook, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	result := make([]*entities.Notebook, 0, limit)
	count := 0
	skipped := 0

	for _, notebook := range r.nbs {
		if notebook.IsDeleted() {
			continue
		}
		titleLower := strings.ToLower(notebook.Title)
		contentLower := strings.ToLower(notebook.Content)
		if !strings.Contains(titleLower, query) && !strings.Contains(contentLower, query) {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, notebook)
		count++
	}

	return result, nil
}

// Delete deletes a notebook by ID
func (r *InMemoryNotebookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.nbs, id)
	return nil
}

// Exists checks if a notebook exists
func (r *InMemoryNotebookRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.nbs[id]
	return exists, nil
}

// Count returns the total count of notebooks
func (r *InMemoryNotebookRepository) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, notebook := range r.nbs {
		if !notebook.IsDeleted() {
			count++
		}
	}
	return count, nil
}

// CountByUserID returns the total count of notebooks for a user
func (r *InMemoryNotebookRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, notebook := range r.nbs {
		if !notebook.IsDeleted() && notebook.UserID == userID {
			count++
		}
	}
	return count, nil
}

// containsTags checks if the notebook contains all the specified tags
func containsTags(notebookTags []string, searchTags []string) bool {
	if len(searchTags) == 0 {
		return true
	}

	tagMap := make(map[string]bool)
	for _, tag := range notebookTags {
		tagMap[strings.ToLower(tag)] = true
	}

	for _, searchTag := range searchTags {
		if !tagMap[strings.ToLower(searchTag)] {
			return false
		}
	}
	return true
}

// String returns a string representation (for debugging)
func (r *InMemoryNotebookRepository) String() string {
	return fmt.Sprintf("InMemoryNotebookRepository(%d notebooks)", len(r.nbs))
}
