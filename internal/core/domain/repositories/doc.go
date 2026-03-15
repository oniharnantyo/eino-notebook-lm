// Package repositories defines interfaces for data persistence.
//
// These interfaces represent the "ports" in Hexagonal Architecture.
// The domain layer defines WHAT data operations are needed, and
// the infrastructure layer provides the implementations ("adapters").
//
// Example:
//
//	type NotebookRepository interface {
//	    Save(ctx context.Context, notebook *entities.Notebook) error
//	    FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error)
//	}
//
// Guidelines:
//   - Define interfaces only (no implementations here)
//   - Return domain entities, not DTOs
//   - Use context.Context for cancellation
//   - Keep methods focused on domain needs
//   - Implementations go in /internal/infrastructure/persistence/
package repositories
