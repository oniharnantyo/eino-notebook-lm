package pgvector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type KnowledgesIndexer struct {
	pool *pgxpool.Pool
}

func NewKnowledgesIndexer(ctx context.Context, pool *pgxpool.Pool) (*KnowledgesIndexer, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	return &KnowledgesIndexer{pool: pool}, nil
}

func (i *KnowledgesIndexer) Store(ctx context.Context, id, sourceID, content string, chunkIndex int, metadata []byte, embedding []float64) error {
	query := `
		INSERT INTO knowledges (id, source_id, content, chunk_index, metadata, embedding)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			chunk_index = EXCLUDED.chunk_index,
			metadata = EXCLUDED.metadata,
			embedding = EXCLUDED.embedding
	`
	_, err := i.pool.Exec(ctx, query, id, sourceID, content, chunkIndex, metadata, embedding)
	return err
}
