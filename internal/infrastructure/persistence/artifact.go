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

// PostgresArtifactRepository implements ArtifactRepository using PostgreSQL
type PostgresArtifactRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresArtifactRepository creates a new PostgreSQL artifact repository
func NewPostgresArtifactRepository(pool *pgxpool.Pool) repositories.ArtifactRepository {
	return &PostgresArtifactRepository{
		pool: pool,
	}
}

// Create creates a new artifact
func (r *PostgresArtifactRepository) Create(ctx context.Context, artifact *entities.Artifact) error {
	query := `
		INSERT INTO artifacts (id, notebook_id, title, type, status, format, content, source_ids, metadata, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.pool.Exec(ctx, query,
		artifact.ID.String(),
		artifact.NotebookID.String(),
		artifact.Title,
		string(artifact.Type),
		string(artifact.Status),
		string(artifact.Format),
		artifact.Content,
		sourceIDsToJSON(artifact.SourceIDs),
		artifactMetadataToJSON(artifact.Metadata),
		artifact.Error,
		artifact.CreatedAt,
		artifact.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}

	return nil
}

// GetByID retrieves an artifact by ID
func (r *PostgresArtifactRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Artifact, error) {
	query := `
		SELECT id, notebook_id, title, type, status, format, content, source_ids, metadata, error, created_at, updated_at, deleted_at
		FROM artifacts
		WHERE id = $1
	`

	var artifact entities.Artifact
	var idStr, notebookIDStr, typeStr, statusStr, formatStr string
	var sourceIDsJSON, metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&idStr,
		&notebookIDStr,
		&artifact.Title,
		&typeStr,
		&statusStr,
		&formatStr,
		&artifact.Content,
		&sourceIDsJSON,
		&metadataJSON,
		&artifact.Error,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
		&artifact.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	// Parse UUIDs
	artifact.ID, _ = uuid.Parse(idStr)
	artifact.NotebookID, _ = uuid.Parse(notebookIDStr)
	artifact.Type = entities.ArtifactType(typeStr)
	artifact.Status = entities.ArtifactStatus(statusStr)
	artifact.Format = entities.ArtifactFormat(formatStr)

	// Parse source IDs
	if sourceIDsJSON != nil {
		if err := json.Unmarshal(sourceIDsJSON, &artifact.SourceIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source_ids: %w", err)
		}
	}

	// Parse metadata
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &artifact, nil
}

// GetByNotebookID retrieves all artifacts for a notebook
func (r *PostgresArtifactRepository) GetByNotebookID(ctx context.Context, notebookID uuid.UUID) ([]*entities.Artifact, error) {
	query := `
		SELECT id, notebook_id, title, type, status, format, content, source_ids, metadata, error, created_at, updated_at, deleted_at
		FROM artifacts
		WHERE notebook_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, _ := r.pool.Query(ctx, query, notebookID.String())
	defer rows.Close()

	var artifacts []*entities.Artifact

	for rows.Next() {
		artifact, err := r.scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// GetByNotebookIDAndType retrieves artifacts of a specific type for a notebook
func (r *PostgresArtifactRepository) GetByNotebookIDAndType(ctx context.Context, notebookID uuid.UUID, artifactType entities.ArtifactType) ([]*entities.Artifact, error) {
	query := `
		SELECT id, notebook_id, title, type, status, format, content, source_ids, metadata, error, created_at, updated_at, deleted_at
		FROM artifacts
		WHERE notebook_id = $1 AND type = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, _ := r.pool.Query(ctx, query, notebookID.String(), string(artifactType))
	defer rows.Close()

	var artifacts []*entities.Artifact

	for rows.Next() {
		artifact, err := r.scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// GetByNotebookIDAndStatus retrieves artifacts with a specific status for a notebook
func (r *PostgresArtifactRepository) GetByNotebookIDAndStatus(ctx context.Context, notebookID uuid.UUID, status entities.ArtifactStatus) ([]*entities.Artifact, error) {
	query := `
		SELECT id, notebook_id, title, type, status, format, content, source_ids, metadata, error, created_at, updated_at, deleted_at
		FROM artifacts
		WHERE notebook_id = $1 AND status = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, _ := r.pool.Query(ctx, query, notebookID.String(), string(status))
	defer rows.Close()

	var artifacts []*entities.Artifact

	for rows.Next() {
		artifact, err := r.scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// Update updates an existing artifact
func (r *PostgresArtifactRepository) Update(ctx context.Context, artifact *entities.Artifact) error {
	query := `
		UPDATE artifacts
		SET title = $2, type = $3, status = $4, format = $5, content = $6, source_ids = $7, metadata = $8, error = $9, updated_at = $10
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		artifact.ID.String(),
		artifact.Title,
		string(artifact.Type),
		string(artifact.Status),
		string(artifact.Format),
		artifact.Content,
		sourceIDsToJSON(artifact.SourceIDs),
		artifactMetadataToJSON(artifact.Metadata),
		artifact.Error,
		artifact.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update artifact: %w", err)
	}

	return nil
}

// Delete soft-deletes an artifact
func (r *PostgresArtifactRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE artifacts SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	return nil
}

// List retrieves artifacts with pagination
func (r *PostgresArtifactRepository) List(ctx context.Context, filter repositories.ArtifactFilter) ([]*entities.Artifact, int, error) {
	// Build WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argCount := 1

	if filter.NotebookID != nil {
		whereClause += fmt.Sprintf(" AND notebook_id = $%d", argCount)
		args = append(args, filter.NotebookID.String())
		argCount++
	}

	if filter.Type != nil {
		whereClause += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, string(*filter.Type))
		argCount++
	}

	if filter.Status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, string(*filter.Status))
		argCount++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM artifacts " + whereClause
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count artifacts: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}

	// Get artifacts
	query := `
		SELECT id, notebook_id, title, type, status, format, content, source_ids, metadata, error, created_at, updated_at, deleted_at
		FROM artifacts
		` + whereClause + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprintf("%d", argCount) + ` OFFSET $` + fmt.Sprintf("%d", argCount+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []*entities.Artifact
	for rows.Next() {
		artifact, err := r.scanArtifact(rows)
		if err != nil {
			return nil, 0, err
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, total, nil
}

// scanArtifact scans an artifact from a database row
func (r *PostgresArtifactRepository) scanArtifact(rows pgx.Rows) (*entities.Artifact, error) {
	var artifact entities.Artifact
	var idStr, notebookIDStr, typeStr, statusStr, formatStr string
	var sourceIDsJSON, metadataJSON []byte

	err := rows.Scan(
		&idStr,
		&notebookIDStr,
		&artifact.Title,
		&typeStr,
		&statusStr,
		&formatStr,
		&artifact.Content,
		&sourceIDsJSON,
		&metadataJSON,
		&artifact.Error,
		&artifact.CreatedAt,
		&artifact.UpdatedAt,
		&artifact.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan artifact: %w", err)
	}

	// Parse UUIDs
	artifact.ID, _ = uuid.Parse(idStr)
	artifact.NotebookID, _ = uuid.Parse(notebookIDStr)
	artifact.Type = entities.ArtifactType(typeStr)
	artifact.Status = entities.ArtifactStatus(statusStr)
	artifact.Format = entities.ArtifactFormat(formatStr)

	// Parse source IDs
	if sourceIDsJSON != nil {
		if err := json.Unmarshal(sourceIDsJSON, &artifact.SourceIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source_ids: %w", err)
		}
	}

	// Parse metadata
	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &artifact, nil
}

// sourceIDsToJSON converts a slice of UUIDs to JSON
func sourceIDsToJSON(sourceIDs []uuid.UUID) []byte {
	if len(sourceIDs) == 0 {
		return nil
	}

	strIDs := make([]string, len(sourceIDs))
	for i, id := range sourceIDs {
		strIDs[i] = id.String()
	}

	data, _ := json.Marshal(strIDs)
	return data
}

// artifactMetadataToJSON converts metadata map to JSON
func artifactMetadataToJSON(metadata map[string]interface{}) []byte {
	if len(metadata) == 0 {
		return nil
	}

	data, _ := json.Marshal(metadata)
	return data
}
