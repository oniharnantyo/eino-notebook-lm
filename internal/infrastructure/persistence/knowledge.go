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
func (r *PostgresKnowledgeRepository) Save(ctx context.Context, k *entities.Knowledge) error {
	query := `
		INSERT INTO knowledges (id, source_id, content, chunk_index, heading_context, first_page, last_page, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			content = EXCLUDED.content,
			chunk_index = EXCLUDED.chunk_index,
			heading_context = EXCLUDED.heading_context,
			first_page = EXCLUDED.first_page,
			last_page = EXCLUDED.last_page,
			metadata = EXCLUDED.metadata,
			created_at = EXCLUDED.created_at
	`

	_, err := r.pool.Exec(ctx, query,
		k.ID.String(),
		k.SourceID.String(),
		k.Content,
		k.ChunkIndex,
		mapToJSON(k.HeadingContext),
		k.FirstPage,
		k.LastPage,
		mapToJSON(k.Metadata),
		k.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save knowledge: %w", err)
	}

	return nil
}

// SaveBatch saves multiple knowledges in a single operation
func (r *PostgresKnowledgeRepository) SaveBatch(ctx context.Context, knowledges []*entities.Knowledge) error {
	batch := &pgx.Batch{}
	query := `
		INSERT INTO knowledges (id, source_id, content, chunk_index, heading_context, first_page, last_page, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			content = EXCLUDED.content,
			chunk_index = EXCLUDED.chunk_index,
			heading_context = EXCLUDED.heading_context,
			first_page = EXCLUDED.first_page,
			last_page = EXCLUDED.last_page,
			metadata = EXCLUDED.metadata,
			created_at = EXCLUDED.created_at
	`

	for _, k := range knowledges {
		batch.Queue(query,
			k.ID.String(),
			k.SourceID.String(),
			k.Content,
			k.ChunkIndex,
			mapToJSON(k.HeadingContext),
			k.FirstPage,
			k.LastPage,
			mapToJSON(k.Metadata),
			k.CreatedAt,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch insert for knowledge at index %d: %w", i, err)
		}
	}

	return nil
}

// FindByID finds a knowledge by ID
func (r *PostgresKnowledgeRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Knowledge, error) {
	query := `
		SELECT id, source_id, content, chunk_index, heading_context, first_page, last_page, metadata, created_at
		FROM knowledges
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id.String())

	var k entities.Knowledge
	var idStr, sourceIDStr string
	var headingJSON, metadataJSON []byte

	err := row.Scan(
		&idStr,
		&sourceIDStr,
		&k.Content,
		&k.ChunkIndex,
		&headingJSON,
		&k.FirstPage,
		&k.LastPage,
		&metadataJSON,
		&k.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find knowledge: %w", err)
	}

	k.ID, _ = uuid.Parse(idStr)
	k.SourceID, _ = uuid.Parse(sourceIDStr)

	if headingJSON != nil {
		json.Unmarshal(headingJSON, &k.HeadingContext)
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &k.Metadata)
	}

	return &k, nil
}

// GetBySourceID retrieves all knowledge chunks for a source
func (r *PostgresKnowledgeRepository) GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Knowledge, error) {
	query := `
		SELECT id, source_id, content, chunk_index, heading_context, first_page, last_page, metadata, created_at
		FROM knowledges
		WHERE source_id = $1
		ORDER BY chunk_index ASC
	`

	rows, err := r.pool.Query(ctx, query, sourceID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledges by source id: %w", err)
	}
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		k, err := r.scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		knowledges = append(knowledges, k)
	}

	return knowledges, nil
}

// FindAll finds all knowledges with pagination
func (r *PostgresKnowledgeRepository) FindAll(ctx context.Context, limit, offset int) ([]*entities.Knowledge, error) {
	query := `
		SELECT id, source_id, content, chunk_index, heading_context, first_page, last_page, metadata, created_at
		FROM knowledges
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find all knowledges: %w", err)
	}
	defer rows.Close()

	var knowledges []*entities.Knowledge

	for rows.Next() {
		k, err := r.scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		knowledges = append(knowledges, k)
	}

	return knowledges, nil
}

// Delete deletes a knowledge by ID
func (r *PostgresKnowledgeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM knowledges WHERE id = $1`

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
	query := `SELECT EXISTS(SELECT 1 FROM knowledges WHERE id = $1)`

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

// FindByIDs finds multiple knowledges by their IDs
func (r *PostgresKnowledgeRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.Knowledge, error) {
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	query := `
		SELECT id, source_id, content, chunk_index, heading_context, first_page, last_page, metadata, created_at
		FROM knowledges
		WHERE id = ANY($1)
	`

	rows, err := r.pool.Query(ctx, query, idStrings)
	if err != nil {
		return nil, fmt.Errorf("failed to find knowledges by ids: %w", err)
	}
	defer rows.Close()

	var knowledges []*entities.Knowledge
	for rows.Next() {
		k, err := r.scanKnowledge(rows)
		if err != nil {
			return nil, err
		}
		knowledges = append(knowledges, k)
	}

	return knowledges, nil
}

// scanKnowledge scans a knowledge from a database row
func (r *PostgresKnowledgeRepository) scanKnowledge(rows pgx.Rows) (*entities.Knowledge, error) {
	var k entities.Knowledge
	var idStr, sourceIDStr string
	var headingJSON, metadataJSON []byte

	err := rows.Scan(
		&idStr,
		&sourceIDStr,
		&k.Content,
		&k.ChunkIndex,
		&headingJSON,
		&k.FirstPage,
		&k.LastPage,
		&metadataJSON,
		&k.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan knowledge: %w", err)
	}

	k.ID, _ = uuid.Parse(idStr)
	k.SourceID, _ = uuid.Parse(sourceIDStr)

	if headingJSON != nil {
		json.Unmarshal(headingJSON, &k.HeadingContext)
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &k.Metadata)
	}

	return &k, nil
}
