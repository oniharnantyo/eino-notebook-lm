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
		INSERT INTO knowledges (knowledge_id, source_id, title, content, source_type, metadata, sub_indexes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (knowledge_id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			source_type = EXCLUDED.source_type,
			metadata = EXCLUDED.metadata,
			sub_indexes = EXCLUDED.sub_indexes
	`

	_, err := r.pool.Exec(ctx, query,
		knowledge.KnowledgeID.String(),
		knowledge.SourceID.String(),
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
		SELECT knowledge_id, source_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		WHERE knowledge_id = $1
	`

	var knowledge entities.Knowledge
	var knowledgeIDStr, sourceIDStr, sourceTypeStr string
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&knowledgeIDStr,
		&sourceIDStr,
		&knowledge.Title,
		&knowledge.Content,
		&sourceTypeStr,
		&metadataJSON,
		&knowledge.SubIndexes,
		&knowledge.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge: %w", err)
	}

	// Parse UUIDs
	knowledge.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
	knowledge.SourceID, _ = uuid.Parse(sourceIDStr)
	knowledge.SourceType = entities.KnowledgeSource(sourceTypeStr)

	// Parse metadata
	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &knowledge.Metadata)
	}

	return &knowledge, nil
}

// GetBySourceID retrieves all knowledge chunks for a source
func (r *PostgresKnowledgeRepository) GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Knowledge, error) {
	query := `
		SELECT knowledge_id, source_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		WHERE source_id = $1
		ORDER BY created_at ASC
	`

	rows, _ := r.pool.Query(ctx, query, sourceID.String())
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		knowledge, err := r.scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		knowledges = append(knowledges, knowledge)
	}

	return knowledges, nil
}

// FindAll finds all knowledges with pagination
func (r *PostgresKnowledgeRepository) FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error) {
	query := `
		SELECT knowledge_id, source_id, title, content, source_type, metadata, sub_indexes, created_at
		FROM knowledges
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, _ := r.pool.Query(ctx, query, limit, offset)
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		knowledge, err := r.scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		knowledges = append(knowledges, knowledge)
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

// DeleteBySourceID deletes all knowledges for a source
func (r *PostgresKnowledgeRepository) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	query := `DELETE FROM knowledges WHERE source_id = $1`

	_, err := r.pool.Exec(ctx, query, sourceID.String())
	if err != nil {
		return fmt.Errorf("failed to delete knowledges by source: %w", err)
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

// CountBySourceID returns the number of knowledge chunks for a source
func (r *PostgresKnowledgeRepository) CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM knowledges WHERE source_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, sourceID.String()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count knowledges by source: %w", err)
	}

	return count, nil
}

// scanKnowledge scans a knowledge from a database row
func (r *PostgresKnowledgeRepository) scanKnowledge(rows pgx.Rows) (*entities.Knowledge, error) {
	var knowledge entities.Knowledge
	var knowledgeIDStr, sourceIDStr, sourceTypeStr string
	var metadataJSON []byte

	err := rows.Scan(
		&knowledgeIDStr,
		&sourceIDStr,
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
	knowledge.SourceID, _ = uuid.Parse(sourceIDStr)
	knowledge.SourceType = entities.KnowledgeSource(sourceTypeStr)

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &knowledge.Metadata)
	}

	return &knowledge, nil
}

// metadataToJSON converts metadata map to JSON
func metadataToJSON(metadata map[string]any) []byte {
	if metadata == nil {
		return nil
	}

	data, _ := json.Marshal(metadata)
	return data
}
