package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// PostgresKnowledgeRepository implements KnowledgeRepository using PostgreSQL
type PostgresKnowledgeRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresKnowledgeRepository creates a new PostgreSQL knowledge repository
func NewPostgresKnowledgeRepository(pool *pgxpool.Pool) repositories.KnowledgeRepository {
	return &PostgresKnowledgeRepository{
		pool: pool,
	}
}

// Save saves a knowledge (create or update)
func (r *PostgresKnowledgeRepository) Save(ctx context.Context, knowledge *entities.Knowledge) error {
	query := `
		INSERT INTO knowledges (knowledge_id, notebook_id, title, content, source_type, metadata, sub_indexes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (knowledge_id) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			source_type = EXCLUDED.source_type,
			metadata = EXCLUDED.metadata,
			sub_indexes = EXCLUDED.sub_indexes
	`

	_, err := r.pool.Exec(ctx, query,
		knowledge.KnowledgeID.String(),
		knowledge.NotebookID.String(),
		knowledge.Title,
		knowledge.Content,
		string(knowledge.SourceType),
		metadataToJSON(knowledge.Metadata),
		knowledge.SubIndexes,
	)

	if err != nil {
		return fmt.Errorf("failed to save knowledge: %w", err)
	}

	return nil
}

// FindByID finds a knowledge by ID
func (r *PostgresKnowledgeRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Knowledge, error) {
	query := `
		SELECT knowledge_id, notebook_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		WHERE knowledge_id = $1
	`

	var knowledge entities.Knowledge
	var knowledgeIDStr, notebookIDStr, sourceTypeStr string
	var metadataJSON []byte
	var createdAtStr string

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&knowledgeIDStr,
		&notebookIDStr,
		&knowledge.Title,
		&knowledge.Content,
		&sourceTypeStr,
		&metadataJSON,
		&knowledge.SubIndexes,
		&createdAtStr,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge: %w", err)
	}

	// Parse UUIDs
	knowledge.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
	knowledge.NotebookID, _ = uuid.Parse(notebookIDStr)
	knowledge.SourceType = entities.KnowledgeSource(sourceTypeStr)

	// Parse metadata
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &knowledge.Metadata)
	}

	return &knowledge, nil
}

// FindByNotebookID finds knowledges by notebook ID with pagination
func (r *PostgresKnowledgeRepository) FindByNotebookID(ctx context.Context, notebookID uuid.UUID, limit, offset int) ([]*entities.Knowledge, error) {
	query := `
		SELECT knowledge_id, notebook_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		WHERE notebook_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, _ := r.pool.Query(ctx, query, notebookID.String(), limit, offset)
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		var knowledge entities.Knowledge
		var knowledgeIDStr, notebookIDStr, sourceTypeStr string
		var metadataJSON []byte

		err := rows.Scan(
			&knowledgeIDStr,
			&notebookIDStr,
			&knowledge.Title,
			&knowledge.Content,
			&sourceTypeStr,
			&metadataJSON,
			&knowledge.SubIndexes,
			&knowledge.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan knowledge: %w", err)
		}

		knowledge.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
		knowledge.NotebookID, _ = uuid.Parse(notebookIDStr)
		knowledge.SourceType = entities.KnowledgeSource(sourceTypeStr)

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &knowledge.Metadata)
		}

		knowledges = append(knowledges, &knowledge)
	}

	return knowledges, nil
}

// FindByNotebookIDAndSourceType finds knowledges by notebook ID and source type
func (r *PostgresKnowledgeRepository) FindByNotebookIDAndSourceType(ctx context.Context, notebookID uuid.UUID, sourceType string, limit, offset int) ([]*entities.Knowledge, error) {
	query := `
		SELECT knowledge_id, notebook_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		WHERE notebook_id = $1 AND source_type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, _ := r.pool.Query(ctx, query, notebookID.String(), sourceType, limit, offset)
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		var knowledge entities.Knowledge
		var knowledgeIDStr, notebookIDStr, sourceTypeStr string
		var metadataJSON []byte

		err := rows.Scan(
			&knowledgeIDStr,
			&notebookIDStr,
			&knowledge.Title,
			&knowledge.Content,
			&sourceTypeStr,
			&metadataJSON,
			&knowledge.SubIndexes,
			&knowledge.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan knowledge: %w", err)
		}

		knowledge.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
		knowledge.NotebookID, _ = uuid.Parse(notebookIDStr)
		knowledge.SourceType = entities.KnowledgeSource(sourceTypeStr)

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &knowledge.Metadata)
		}

		knowledges = append(knowledges, &knowledge)
	}

	return knowledges, nil
}

// FindAll finds all knowledges with pagination
func (r *PostgresKnowledgeRepository) FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error) {
	query := `
		SELECT knowledge_id, notebook_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, _ := r.pool.Query(ctx, query, limit, offset)
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		var knowledge entities.Knowledge
		var knowledgeIDStr, notebookIDStr, sourceTypeStr string
		var metadataJSON []byte

		err := rows.Scan(
			&knowledgeIDStr,
			&notebookIDStr,
			&knowledge.Title,
			&knowledge.Content,
			&sourceTypeStr,
			&metadataJSON,
			&knowledge.SubIndexes,
			&knowledge.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan knowledge: %w", err)
		}

		knowledge.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
		knowledge.NotebookID, _ = uuid.Parse(notebookIDStr)
		knowledge.SourceType = entities.KnowledgeSource(sourceTypeStr)

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &knowledge.Metadata)
		}

		knowledges = append(knowledges, &knowledge)
	}

	return knowledges, nil
}

// Delete deletes a knowledge by ID
func (r *PostgresKnowledgeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM knowledges WHERE knowledge_id = $1`

	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete knowledge: %w", err)
	}

	return nil
}

// DeleteByNotebookID deletes all knowledges for a notebook
func (r *PostgresKnowledgeRepository) DeleteByNotebookID(ctx context.Context, notebookID uuid.UUID) error {
	query := `DELETE FROM knowledges WHERE notebook_id = $1`

	_, err := r.pool.Exec(ctx, query, notebookID.String())
	if err != nil {
		return fmt.Errorf("failed to delete knowledges by notebook: %w", err)
	}

	return nil
}

// Exists checks if a knowledge exists
func (r *PostgresKnowledgeRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM knowledges WHERE knowledge_id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, id.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check knowledge existence: %w", err)
	}

	return exists, nil
}

// Count returns the total count of knowledges
func (r *PostgresKnowledgeRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM knowledges`

	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count knowledges: %w", err)
	}

	return count, nil
}

// CountByNotebookID returns the total count of knowledges for a notebook
func (r *PostgresKnowledgeRepository) CountByNotebookID(ctx context.Context, notebookID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM knowledges WHERE notebook_id = $1`

	var count int64
	err := r.pool.QueryRow(ctx, query, notebookID.String()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count knowledges by notebook: %w", err)
	}

	return count, nil
}

// metadataToJSON converts metadata map to JSON
func metadataToJSON(metadata map[string]interface{}) []byte {
	if metadata == nil {
		return nil
	}

	data, _ := json.Marshal(metadata)
	return data
}
