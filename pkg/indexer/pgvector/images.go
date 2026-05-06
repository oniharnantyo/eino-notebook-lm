package pgvector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImagesIndexer struct {
	pool *pgxpool.Pool
}

func NewImagesIndexer(ctx context.Context, pool *pgxpool.Pool) (*ImagesIndexer, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	return &ImagesIndexer{pool: pool}, nil
}

func (i *ImagesIndexer) Store(ctx context.Context, id, sourceID, s3Key, format string, width, height, pageNumber int, description string, metadata []byte, embedding []float64) error {
	query := `
		INSERT INTO images (id, source_id, s3_key, format, width, height, page_number, description, metadata, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			s3_key = EXCLUDED.s3_key,
			format = EXCLUDED.format,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			page_number = EXCLUDED.page_number,
			description = EXCLUDED.description,
			metadata = EXCLUDED.metadata,
			embedding = EXCLUDED.embedding
	`
	_, err := i.pool.Exec(ctx, query, id, sourceID, s3Key, format, width, height, pageNumber, description, metadata, embedding)
	return err
}
