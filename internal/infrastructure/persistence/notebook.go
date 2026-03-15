package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/valueobjects"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// PostgresNotebookRepository implements NotebookRepository using PostgreSQL
type PostgresNotebookRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresNotebookRepository creates a new PostgreSQL notebook repository
func NewPostgresNotebookRepository(pool *pgxpool.Pool) repositories.NotebookRepository {
	return &PostgresNotebookRepository{
		pool: pool,
	}
}

// Save saves a notebook (create or update)
func (r *PostgresNotebookRepository) Save(ctx context.Context, notebook *entities.Notebook) error {
	query := `
		INSERT INTO notebooks (id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			status = EXCLUDED.status,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at
	`

	_, err := r.pool.Exec(ctx, query,
		notebook.ID.String(),
		notebook.UserID.String(),
		notebook.Title,
		notebook.Description,
		notebook.Content,
		notebook.Status.String(),
		notebook.Tags,
		metadataToJSON(notebook.Metadata),
		notebook.CreatedAt,
		notebook.UpdatedAt,
		notebook.DeletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save notebook: %w", err)
	}

	return nil
}

// FindByID finds a notebook by ID
func (r *PostgresNotebookRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error) {
	query := `
		SELECT id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at
		FROM notebooks
		WHERE id = $1
	`

	var notebook entities.Notebook
	var idStr, userIDStr, statusStr string
	var tagsJSON []byte
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&idStr,
		&userIDStr,
		&notebook.Title,
		&notebook.Description,
		&notebook.Content,
		&statusStr,
		&tagsJSON,
		&metadataJSON,
		&notebook.CreatedAt,
		&notebook.UpdatedAt,
		&notebook.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find notebook: %w", err)
	}

	// Parse UUIDs
	notebook.ID, _ = uuid.Parse(idStr)
	notebook.UserID, _ = uuid.Parse(userIDStr)
	notebook.Status = parseStatus(statusStr)

	// Parse JSON fields
	if tagsJSON != nil {
		json.Unmarshal(tagsJSON, &notebook.Tags)
	}
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &notebook.Metadata)
	}

	return &notebook, nil
}

// FindAll finds all notebooks with pagination
func (r *PostgresNotebookRepository) FindAll(ctx context.Context, limit, offset int) ([]*entities.Notebook, error) {
	query := `
		SELECT id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at
		FROM notebooks
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, _ := r.pool.Query(ctx, query, limit, offset)
	defer rows.Close()

	var notebooks []*entities.Notebook

	for rows.Next() {
		notebook, err := r.scanNotebook(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notebook: %w", err)
		}
		notebooks = append(notebooks, notebook)
	}

	return notebooks, nil
}

// FindByUserID finds notebooks by user ID with pagination
func (r *PostgresNotebookRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Notebook, error) {
	query := `
		SELECT id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at
		FROM notebooks
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, _ := r.pool.Query(ctx, query, userID.String(), limit, offset)
	defer rows.Close()

	var notebooks []*entities.Notebook

	for rows.Next() {
		notebook, err := r.scanNotebook(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notebook: %w", err)
		}
		notebooks = append(notebooks, notebook)
	}

	return notebooks, nil
}

// FindByStatus finds notebooks by status
func (r *PostgresNotebookRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*entities.Notebook, error) {
	query := `
		SELECT id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at
		FROM notebooks
		WHERE status = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, _ := r.pool.Query(ctx, query, status, limit, offset)
	defer rows.Close()

	var notebooks []*entities.Notebook

	for rows.Next() {
		notebook, err := r.scanNotebook(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notebook: %w", err)
		}
		notebooks = append(notebooks, notebook)
	}

	return notebooks, nil
}

// FindByTags finds notebooks by tags
func (r *PostgresNotebookRepository) FindByTags(ctx context.Context, tags []string, limit, offset int) ([]*entities.Notebook, error) {
	query := `
		SELECT id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at
		FROM notebooks
		WHERE deleted_at IS NULL AND tags @> $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	tagsJSON, _ := json.Marshal(tags)

	rows, _ := r.pool.Query(ctx, query, tagsJSON, limit, offset)
	defer rows.Close()

	var notebooks []*entities.Notebook

	for rows.Next() {
		notebook, err := r.scanNotebook(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notebook: %w", err)
		}
		notebooks = append(notebooks, notebook)
	}

	return notebooks, nil
}

// Search searches notebooks by title or content
func (r *PostgresNotebookRepository) Search(ctx context.Context, query string, limit, offset int) ([]*entities.Notebook, error) {
	searchQuery := fmt.Sprintf("%%%s%%", strings.ToLower(query))

	sqlQuery := `
		SELECT id, user_id, title, description, content, status, tags, metadata, created_at, updated_at, deleted_at
		FROM notebooks
		WHERE deleted_at IS NULL
		AND (LOWER(title) LIKE $1 OR LOWER(content) LIKE $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, _ := r.pool.Query(ctx, sqlQuery, searchQuery, limit, offset)
	defer rows.Close()

	var notebooks []*entities.Notebook

	for rows.Next() {
		notebook, err := r.scanNotebook(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notebook: %w", err)
		}
		notebooks = append(notebooks, notebook)
	}

	return notebooks, nil
}

// Delete deletes a notebook by ID
func (r *PostgresNotebookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notebooks WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete notebook: %w", err)
	}

	return nil
}

// Exists checks if a notebook exists
func (r *PostgresNotebookRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM notebooks WHERE id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, id.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check notebook existence: %w", err)
	}

	return exists, nil
}

// Count returns the total count of notebooks
func (r *PostgresNotebookRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM notebooks WHERE deleted_at IS NULL`

	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count notebooks: %w", err)
	}

	return count, nil
}

// CountByUserID returns the total count of notebooks for a user
func (r *PostgresNotebookRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM notebooks WHERE user_id = $1 AND deleted_at IS NULL`

	var count int64
	err := r.pool.QueryRow(ctx, query, userID.String()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count notebooks by user: %w", err)
	}

	return count, nil
}

// scanNotebook scans a row into a Notebook entity
func (r *PostgresNotebookRepository) scanNotebook(rows interface{}) (*entities.Notebook, error) {
	type scanner interface {
		Scan(dest ...interface{}) error
	}

	var notebook entities.Notebook
	var idStr, userIDStr, statusStr string
	var tagsJSON []byte
	var metadataJSON []byte

	err := rows.(scanner).Scan(
		&idStr,
		&userIDStr,
		&notebook.Title,
		&notebook.Description,
		&notebook.Content,
		&statusStr,
		&tagsJSON,
		&metadataJSON,
		&notebook.CreatedAt,
		&notebook.UpdatedAt,
		&notebook.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse UUIDs
	notebook.ID, _ = uuid.Parse(idStr)
	notebook.UserID, _ = uuid.Parse(userIDStr)
	notebook.Status = parseStatus(statusStr)

	// Parse JSON fields
	if tagsJSON != nil {
		json.Unmarshal(tagsJSON, &notebook.Tags)
	}
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &notebook.Metadata)
	}

	return &notebook, nil
}

// parseStatus parses status string to NotebookStatus
func parseStatus(status string) valueobjects.NotebookStatus {
	switch valueobjects.NotebookStatus(status) {
	case valueobjects.StatusActive, valueobjects.StatusArchived, valueobjects.StatusDeleted:
		return valueobjects.NotebookStatus(status)
	default:
		return valueobjects.StatusActive
	}
}
