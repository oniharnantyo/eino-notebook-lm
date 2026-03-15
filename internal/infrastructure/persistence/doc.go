// Package persistence contains repository implementations.
//
// These are the "adapters" in Hexagonal Architecture that implement
// the repository interfaces defined in the domain layer.
//
// Example:
//
//	type PostgresNotebookRepository struct {
//	    pool *pgxpool.Pool
//	}
//
//	func (r *PostgresNotebookRepository) Save(ctx context.Context, notebook *entities.Notebook) error {
//	    _, err := r.pool.Exec(ctx, "INSERT INTO notebooks ...")
//	    return err
//	}
//
// Guidelines:
//   - Implement domain repository interfaces
//   - Handle database-specific concerns
//   - Return domain errors, not DB errors
//   - Use connection pooling
//   - Implementations can be swapped (e.g., for testing)
package persistence
