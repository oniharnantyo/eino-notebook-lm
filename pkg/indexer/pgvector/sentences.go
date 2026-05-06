package pgvector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SentencesIndexer struct {
	pool *pgxpool.Pool
}

func NewSentencesIndexer(ctx context.Context, pool *pgxpool.Pool) (*SentencesIndexer, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	return &SentencesIndexer{pool: pool}, nil
}

func (i *SentencesIndexer) Store(ctx context.Context, id, knowledgeID, content string, position int, metadata []byte, embedding []float64) error {
	query := `
		INSERT INTO sentences (id, knowledge_id, content, position, metadata, embedding)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			position = EXCLUDED.position,
			metadata = EXCLUDED.metadata,
			embedding = EXCLUDED.embedding
	`
	_, err := i.pool.Exec(ctx, query, id, knowledgeID, content, position, metadata, embedding)
	return err
}
