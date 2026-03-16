package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// PostgresSourceRepository implements SourceRepository using PostgreSQL
type PostgresSourceRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSourceRepository creates a new PostgreSQL source repository
func NewPostgresSourceRepository(pool *pgxpool.Pool) repositories.SourceRepository {
	return &PostgresSourceRepository{
		pool: pool,
	}
}

// Create creates a new source
func (r *PostgresSourceRepository) Create(ctx context.Context, source *entities.Source) error {
	query := `
		INSERT INTO sources (id, notebook_id, title, uri, content_type, content, chunk_count, total_size, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		source.ID.String(),
		source.NotebookID.String(),
		source.Title,
		source.URI,
		string(source.ContentType),
		source.Content,
		source.ChunkCount,
		source.TotalSize,
		metadataToJSON(source.Metadata),
		source.CreatedAt,
		source.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}

	return nil
}

// GetByID retrieves a source by ID
func (r *PostgresSourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Source, error) {
	query := `
		SELECT id, notebook_id, title, uri, content_type, content, chunk_count, total_size, metadata, created_at, updated_at, deleted_at
		FROM sources
		WHERE id = $1
	`

	var source entities.Source
	var idStr, notebookIDStr, contentTypeStr string
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&idStr,
		&notebookIDStr,
		&source.Title,
		&source.URI,
		&contentTypeStr,
		&source.Content,
		&source.ChunkCount,
		&source.TotalSize,
		&metadataJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
		&source.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	// Parse UUIDs
	source.ID, _ = uuid.Parse(idStr)
	source.NotebookID, _ = uuid.Parse(notebookIDStr)
	source.ContentType = entities.ContentType(contentTypeStr)

	// Parse metadata
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &source.Metadata)
	}

	return &source, nil
}

// GetByNotebookID retrieves all sources for a notebook
func (r *PostgresSourceRepository) GetByNotebookID(ctx context.Context, notebookID uuid.UUID) ([]*entities.Source, error) {
	query := `
		SELECT id, notebook_id, title, uri, content_type, content, chunk_count, total_size, metadata, created_at, updated_at, deleted_at
		FROM sources
		WHERE notebook_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, _ := r.pool.Query(ctx, query, notebookID.String())
	defer rows.Close()

	var sources []*entities.Source

	for rows.Next() {
		source, err := r.scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// GetByURI retrieves a source by its URI
func (r *PostgresSourceRepository) GetByURI(ctx context.Context, notebookID uuid.UUID, uri string) (*entities.Source, error) {
	query := `
		SELECT id, notebook_id, title, uri, content_type, content, chunk_count, total_size, metadata, created_at, updated_at, deleted_at
		FROM sources
		WHERE notebook_id = $1 AND uri = $2 AND deleted_at IS NULL
		LIMIT 1
	`

	var source entities.Source
	var idStr, notebookIDStr, contentTypeStr string
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, notebookID.String(), uri).Scan(
		&idStr,
		&notebookIDStr,
		&source.Title,
		&source.URI,
		&contentTypeStr,
		&source.Content,
		&source.ChunkCount,
		&source.TotalSize,
		&metadataJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
		&source.DeletedAt,
	)

	if err != nil {
		return nil, nil // Not found
	}

	// Parse UUIDs
	source.ID, _ = uuid.Parse(idStr)
	source.NotebookID, _ = uuid.Parse(notebookIDStr)
	source.ContentType = entities.ContentType(contentTypeStr)

	// Parse metadata
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &source.Metadata)
	}

	return &source, nil
}

// Update updates an existing source
func (r *PostgresSourceRepository) Update(ctx context.Context, source *entities.Source) error {
	query := `
		UPDATE sources
		SET title = $2, uri = $3, content_type = $4, content = $5, chunk_count = $6, total_size = $7, metadata = $8, updated_at = $9
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		source.ID.String(),
		source.Title,
		source.URI,
		string(source.ContentType),
		source.Content,
		source.ChunkCount,
		source.TotalSize,
		metadataToJSON(source.Metadata),
		source.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}

	return nil
}

// Delete soft-deletes a source
func (r *PostgresSourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sources SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}

	return nil
}

// List retrieves sources with pagination
func (r *PostgresSourceRepository) List(ctx context.Context, filter repositories.SourceFilter) ([]*entities.Source, int, error) {
	// Build WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argCount := 1

	if filter.NotebookID != nil {
		whereClause += fmt.Sprintf(" AND notebook_id = $%d", argCount)
		args = append(args, filter.NotebookID.String())
		argCount++
	}

	if filter.ContentType != nil {
		whereClause += fmt.Sprintf(" AND content_type = $%d", argCount)
		args = append(args, string(*filter.ContentType))
		argCount++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM sources " + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sources: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
		if orderBy != "created_at DESC" && orderBy != "created_at ASC" &&
			orderBy != "title ASC" && orderBy != "title DESC" &&
			orderBy != "chunk_count ASC" && orderBy != "chunk_count DESC" {
			orderBy = "created_at DESC"
		}
	}

	// Get sources
	query := `
		SELECT id, notebook_id, title, uri, content_type, content, chunk_count, total_size, metadata, created_at, updated_at, deleted_at
		FROM sources
		` + whereClause + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprintf("%d", argCount) + ` OFFSET $` + fmt.Sprintf("%d", argCount+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, _ := r.pool.Query(ctx, query, args...)
	defer rows.Close()

	var sources []*entities.Source
	for rows.Next() {
		source, err := r.scanSource(rows)
		if err != nil {
			return nil, 0, err
		}
		sources = append(sources, source)
	}

	return sources, total, nil
}

// IncrementChunkCount atomically increments the chunk counter
func (r *PostgresSourceRepository) IncrementChunkCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sources SET chunk_count = chunk_count + 1, updated_at = NOW() WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to increment chunk count: %w", err)
	}

	return nil
}

// scanSource scans a source from a database row
func (r *PostgresSourceRepository) scanSource(rows pgx.Rows) (*entities.Source, error) {
	var source entities.Source
	var idStr, notebookIDStr, contentTypeStr string
	var metadataJSON []byte

	err := rows.Scan(
		&idStr,
		&notebookIDStr,
		&source.Title,
		&source.URI,
		&contentTypeStr,
		&source.Content,
		&source.ChunkCount,
		&source.TotalSize,
		&metadataJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
		&source.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan source: %w", err)
	}

	// Parse UUIDs
	source.ID, _ = uuid.Parse(idStr)
	source.NotebookID, _ = uuid.Parse(notebookIDStr)
	source.ContentType = entities.ContentType(contentTypeStr)

	// Parse metadata
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &source.Metadata)
	}

	return &source, nil
}
