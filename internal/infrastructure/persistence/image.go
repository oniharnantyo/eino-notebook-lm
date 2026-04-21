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

// PostgresImageRepository implements ImageRepository using PostgreSQL with pgvector
type PostgresImageRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresImageRepository creates a new PostgreSQL image repository
func NewPostgresImageRepository(pool *pgxpool.Pool) repositories.ImageRepository {
	return &PostgresImageRepository{
		pool: pool,
	}
}

// Save saves an image
func (r *PostgresImageRepository) Save(ctx context.Context, img *entities.Image) error {
	query := `
		INSERT INTO images (id, source_id, s3_key, format, width, height, description, page_number, embedding, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			s3_key = EXCLUDED.s3_key,
			format = EXCLUDED.format,
			width = EXCLUDED.width,
			height = EXCLUDED.height,
			description = EXCLUDED.description,
			page_number = EXCLUDED.page_number,
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata,
			created_at = EXCLUDED.created_at
	`

	_, err := r.pool.Exec(ctx, query,
		img.ID.String(),
		img.SourceID.String(),
		img.S3Key,
		img.Format,
		img.Width,
		img.Height,
		img.Description,
		img.PageNumber,
		float32VectorToString(img.Embedding),
		mapToJSON(img.Metadata),
		img.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}

// FindByID finds an image by ID
func (r *PostgresImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Image, error) {
	query := `
		SELECT id, source_id, s3_key, format, width, height, description, page_number, embedding, metadata, created_at
		FROM images
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id.String())
	return r.scanImageRow(row)
}

// FindBySourceID retrieves all images for a source
func (r *PostgresImageRepository) FindBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Image, error) {
	query := `
		SELECT id, source_id, s3_key, format, width, height, description, page_number, embedding, metadata, created_at
		FROM images
		WHERE source_id = $1
		ORDER BY page_number ASC
	`

	rows, err := r.pool.Query(ctx, query, sourceID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to find images by source id: %w", err)
	}
	defer rows.Close()

	var images []*entities.Image
	for rows.Next() {
		img, err := r.scanImageRow(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, img)
	}

	return images, nil
}

// Delete deletes an image by ID
func (r *PostgresImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM images WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	return nil
}

// DeleteBySourceID deletes all images for a source
func (r *PostgresImageRepository) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	query := `DELETE FROM images WHERE source_id = $1`
	_, err := r.pool.Exec(ctx, query, sourceID.String())
	if err != nil {
		return fmt.Errorf("failed to delete images by source id: %w", err)
	}
	return nil
}

// CountBySourceID returns the number of images for a source
func (r *PostgresImageRepository) CountBySourceID(ctx context.Context, sourceID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM images WHERE source_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, sourceID.String()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count images by source id: %w", err)
	}
	return count, nil
}

// scanImageRow scans an image from a database row
func (r *PostgresImageRepository) scanImageRow(row pgx.Row) (*entities.Image, error) {
	var img entities.Image
	var idStr, sourceIDStr string
	var embeddingStr *string
	var metadataJSON []byte

	err := row.Scan(
		&idStr,
		&sourceIDStr,
		&img.S3Key,
		&img.Format,
		&img.Width,
		&img.Height,
		&img.Description,
		&img.PageNumber,
		&embeddingStr,
		&metadataJSON,
		&img.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan image: %w", err)
	}

	img.ID, _ = uuid.Parse(idStr)
	img.SourceID, _ = uuid.Parse(sourceIDStr)
	if embeddingStr != nil {
		img.Embedding = stringToFloat32Vector(*embeddingStr)
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &img.Metadata)
	}

	return &img, nil
}
