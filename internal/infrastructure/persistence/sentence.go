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

// PostgresSentenceRepository implements SentenceRepository using PostgreSQL with pgvector
type PostgresSentenceRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresSentenceRepository creates a new PostgreSQL sentence repository
func NewPostgresSentenceRepository(pool *pgxpool.Pool) repositories.SentenceRepository {
	return &PostgresSentenceRepository{
		pool: pool,
	}
}

// Save saves a sentence
func (r *PostgresSentenceRepository) Save(ctx context.Context, s *entities.Sentence) error {
	query := `
		INSERT INTO sentences (id, knowledge_id, content, embedding, position, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			knowledge_id = EXCLUDED.knowledge_id,
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			position = EXCLUDED.position,
			metadata = EXCLUDED.metadata,
			created_at = EXCLUDED.created_at
	`

	_, err := r.pool.Exec(ctx, query,
		s.ID.String(),
		s.KnowledgeID.String(),
		s.Content,
		float32VectorToString(s.Embedding),
		s.Position,
		mapToJSON(s.Metadata),
		s.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save sentence: %w", err)
	}

	return nil
}

// SaveBatch saves multiple sentences in a single operation
func (r *PostgresSentenceRepository) SaveBatch(ctx context.Context, sentences []*entities.Sentence) error {
	batch := &pgx.Batch{}
	query := `
		INSERT INTO sentences (id, knowledge_id, content, embedding, position, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`

	for _, s := range sentences {
		batch.Queue(query,
			s.ID.String(),
			s.KnowledgeID.String(),
			s.Content,
			float32VectorToString(s.Embedding),
			s.Position,
			mapToJSON(s.Metadata),
			s.CreatedAt,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("failed to execute batch insert for sentence at index %d: %w", i, err)
		}
	}

	return nil
}

// FindByID finds a sentence by ID
func (r *PostgresSentenceRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Sentence, error) {
	query := `
		SELECT id, knowledge_id, content, embedding, position, metadata, created_at
		FROM sentences
		WHERE id = $1
	`

	var s entities.Sentence
	var idStr, knowledgeIDStr string
	var embeddingStr string
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id.String()).Scan(
		&idStr,
		&knowledgeIDStr,
		&s.Content,
		&embeddingStr,
		&s.Position,
		&metadataJSON,
		&s.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find sentence: %w", err)
	}

	s.ID, _ = uuid.Parse(idStr)
	s.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
	s.Embedding = stringToFloat32Vector(embeddingStr)

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &s.Metadata)
	}

	return &s, nil
}

// FindByKnowledgeID retrieves all sentences for a knowledge chunk
func (r *PostgresSentenceRepository) FindByKnowledgeID(ctx context.Context, knowledgeID uuid.UUID) ([]*entities.Sentence, error) {
	query := `
		SELECT id, knowledge_id, content, embedding, position, metadata, created_at
		FROM sentences
		WHERE knowledge_id = $1
		ORDER BY position ASC
	`

	rows, err := r.pool.Query(ctx, query, knowledgeID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find sentences by knowledge id: %w", err)
	}
	defer rows.Close()

	var sentences []*entities.Sentence
	for rows.Next() {
		s, err := r.scanSentence(rows)
		if err != nil {
			return nil, err
		}
		sentences = append(sentences, s)
	}

	return sentences, nil
}

// Delete deletes a sentence by ID
func (r *PostgresSentenceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sentences WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete sentence: %w", err)
	}
	return nil
}

// DeleteByKnowledgeID deletes all sentences for a knowledge chunk
func (r *PostgresSentenceRepository) DeleteByKnowledgeID(ctx context.Context, knowledgeID uuid.UUID) error {
	query := `DELETE FROM sentences WHERE knowledge_id = $1`
	_, err := r.pool.Exec(ctx, query, knowledgeID.String())
	if err != nil {
		return fmt.Errorf("failed to delete sentences by knowledge id: %w", err)
	}
	return nil
}

// DeleteBySourceID deletes all sentences associated with a source
func (r *PostgresSentenceRepository) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	query := `
		DELETE FROM sentences
		WHERE knowledge_id IN (
			SELECT id FROM knowledges WHERE source_id = $1
		)
	`
	_, err := r.pool.Exec(ctx, query, sourceID.String())
	if err != nil {
		return fmt.Errorf("failed to delete sentences by source id: %w", err)
	}
	return nil
}

// scanSentence scans a sentence from a database row
func (r *PostgresSentenceRepository) scanSentence(rows pgx.Rows) (*entities.Sentence, error) {
	var s entities.Sentence
	var idStr, knowledgeIDStr string
	var embeddingStr string
	var metadataJSON []byte

	err := rows.Scan(
		&idStr,
		&knowledgeIDStr,
		&s.Content,
		&embeddingStr,
		&s.Position,
		&metadataJSON,
		&s.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan sentence: %w", err)
	}

	s.ID, _ = uuid.Parse(idStr)
	s.KnowledgeID, _ = uuid.Parse(knowledgeIDStr)
	s.Embedding = stringToFloat32Vector(embeddingStr)

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &s.Metadata)
	}

	return &s, nil
}
