/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pgvector

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Indexer is an implementation of indexer.Indexer using PostgreSQL with pgvector.
type Indexer struct {
	config *Config
	pool   *pgxpool.Pool
}

// NewIndexer creates a new pgvector indexer.
func NewIndexer(ctx context.Context, config *Config) (*Indexer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.Pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	config.setDefaults()

	idx := &Indexer{
		config: config,
		pool:   config.Pool,
	}

	// Create table if auto-create is enabled
	if config.AutoCreateTable {
		if err := idx.createTable(ctx); err != nil {
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create index if enabled
	if config.CreateIndexIfNotExists {
		if err := idx.createIndex(ctx); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	}

	return idx, nil
}

// Store stores documents and returns their assigned IDs.
// Implements indexer.Indexer.
func (i *Indexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}

	// Get common options
	commonOpts := indexer.GetCommonOptions(nil, opts...)
	storeOpts := getStoreOptions(opts...)

	// Generate embeddings if an embedder is provided
	var embeddings [][]float64
	var err error

	if commonOpts.Embedding != nil {
		texts := make([]string, len(docs))
		for j, doc := range docs {
			texts[j] = doc.Content
		}
		embeddings, err = commonOpts.Embedding.EmbedStrings(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}
	}

	// Prepare document IDs and data
	ids := make([]string, len(docs))
	for j, doc := range docs {
		if doc.ID == "" {
			ids[j] = uuid.New().String()
		} else {
			ids[j] = doc.ID
		}
	}

	// Build the query based on options
	if storeOpts.Upsert {
		return i.storeUpsert(ctx, docs, ids, embeddings, commonOpts)
	}

	if storeOpts.SkipExisting {
		return i.storeSkipExisting(ctx, docs, ids, embeddings, commonOpts)
	}

	// Default: simple insert
	return i.storeInsert(ctx, docs, ids, embeddings, commonOpts)
}

// storeInsert performs a simple insert operation.
func (i *Indexer) storeInsert(ctx context.Context, docs []*schema.Document, ids []string, embeddings [][]float64, opts *indexer.Options) ([]string, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, $5)
	`, i.config.TableName, i.config.IDColumn, i.config.ContentColumn,
		i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn)

	for j, doc := range docs {
		_, err := i.pool.Exec(ctx, query,
			ids[j],
			doc.Content,
			vectorToString(embeddings, j),
			metadataToJSONB(doc.MetaData),
			subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert document %s: %w", ids[j], err)
		}
	}

	return ids, nil
}

// storeUpsert performs an upsert operation (insert or update on conflict).
func (i *Indexer) storeUpsert(ctx context.Context, docs []*schema.Document, ids []string, embeddings [][]float64, opts *indexer.Options) ([]string, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (%s) DO UPDATE SET
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s,
			%s = EXCLUDED.%s
	`, i.config.TableName,
		i.config.IDColumn, i.config.ContentColumn,
		i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn,
		i.config.IDColumn,
		i.config.ContentColumn, i.config.ContentColumn,
		i.config.EmbeddingColumn, i.config.EmbeddingColumn,
		i.config.MetadataColumn, i.config.MetadataColumn,
		i.config.SubIndexesColumn, i.config.SubIndexesColumn,
	)

	for j, doc := range docs {
		_, err := i.pool.Exec(ctx, query,
			ids[j],
			doc.Content,
			vectorToString(embeddings, j),
			metadataToJSONB(doc.MetaData),
			subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert document %s: %w", ids[j], err)
		}
	}

	return ids, nil
}

// storeSkipExisting inserts only documents that don't already exist.
func (i *Indexer) storeSkipExisting(ctx context.Context, docs []*schema.Document, ids []string, embeddings [][]float64, opts *indexer.Options) ([]string, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s, %s)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (%s) DO NOTHING
	`, i.config.TableName,
		i.config.IDColumn, i.config.ContentColumn,
		i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn,
		i.config.IDColumn,
	)

	resultIds := make([]string, 0, len(ids))

	for j, doc := range docs {
		result, err := i.pool.Exec(ctx, query,
			ids[j],
			doc.Content,
			vectorToString(embeddings, j),
			metadataToJSONB(doc.MetaData),
			subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert document %s: %w", ids[j], err)
		}

		// Only add ID if the row was actually inserted
		if result.RowsAffected() > 0 {
			resultIds = append(resultIds, ids[j])
		}
	}

	return resultIds, nil
}

// createTable creates the documents table if it doesn't exist.
func (i *Indexer) createTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s TEXT PRIMARY KEY,
			%s TEXT NOT NULL,
			%s vector(%d),
			%s JSONB,
			%s TEXT[],
			created_at TIMESTAMP DEFAULT NOW()
		)
	`, i.config.TableName,
		i.config.IDColumn,
		i.config.ContentColumn,
		i.config.EmbeddingColumn,
		i.config.Dimension,
		i.config.MetadataColumn,
		i.config.SubIndexesColumn,
	)

	_, err := i.pool.Exec(ctx, query)
	return err
}

// createIndex creates a HNSW index on the embedding column if it doesn't exist.
func (i *Indexer) createIndex(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s ON %s
		USING hnsw (%s %s)
	`, i.config.IndexName, i.config.TableName,
		i.config.EmbeddingColumn, i.config.DistanceFunction.indexOperator())

	_, err := i.pool.Exec(ctx, query)
	return err
}

// vectorToString converts a vector to PostgreSQL vector format.
func vectorToString(embeddings [][]float64, index int) interface{} {
	if embeddings == nil || len(embeddings) <= index || embeddings[index] == nil {
		return nil
	}

	vec := embeddings[index]
	if len(vec) == 0 {
		return nil
	}

	// Convert to pgvector format: "[x1,x2,x3,...]"
	str := "["
	for i, v := range vec {
		if i > 0 {
			str += ","
		}
		str += fmt.Sprintf("%g", v)
	}
	str += "]"

	return str
}

// metadataToJSONB converts metadata map to JSONB.
func metadataToJSONB(metadata map[string]any) interface{} {
	if metadata == nil {
		return nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}

	return data
}

// subIndexesToArray merges option sub-indexes with document sub-indexes.
func subIndexesToArray(optionIndexes []string, docIndexes []string) interface{} {
	// Create a set to avoid duplicates
	indexSet := make(map[string]struct{})

	for _, idx := range optionIndexes {
		if idx != "" {
			indexSet[idx] = struct{}{}
		}
	}

	for _, idx := range docIndexes {
		if idx != "" {
			indexSet[idx] = struct{}{}
		}
	}

	if len(indexSet) == 0 {
		return nil
	}

	result := make([]string, 0, len(indexSet))
	for idx := range indexSet {
		result = append(result, idx)
	}

	return result
}

// GetPool returns the underlying connection pool.
func (i *Indexer) GetPool() *pgxpool.Pool {
	return i.pool
}

// GetConfig returns the indexer configuration.
func (i *Indexer) GetConfig() *Config {
	return i.config
}

// CreateExtension creates the vector extension in the database.
func (i *Indexer) CreateExtension(ctx context.Context) error {
	_, err := i.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	return err
}

// DropTable drops the documents table.
func (i *Indexer) DropTable(ctx context.Context) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", i.config.TableName)
	_, err := i.pool.Exec(ctx, query)
	return err
}

// DropIndex drops the embedding index.
func (i *Indexer) DropIndex(ctx context.Context) error {
	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", i.config.IndexName)
	_, err := i.pool.Exec(ctx, query)
	return err
}

// TableExists checks if the documents table exists.
func (i *Indexer) TableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`

	var exists bool
	err := i.pool.QueryRow(ctx, query, i.config.TableName).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// DocumentExists checks if a document with the given ID exists.
func (i *Indexer) DocumentExists(ctx context.Context, id string) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", i.config.TableName, i.config.IDColumn)

	var exists bool
	err := i.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetDocument retrieves a document by ID.
func (i *Indexer) GetDocument(ctx context.Context, id string) (*schema.Document, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s
		FROM %s
		WHERE %s = $1
	`, i.config.IDColumn, i.config.ContentColumn, i.config.EmbeddingColumn, i.config.MetadataColumn,
		i.config.TableName, i.config.IDColumn)

	var docID, content string
	var embeddingStr []byte
	var metadataJSONB []byte

	err := i.pool.QueryRow(ctx, query, id).Scan(&docID, &content, &embeddingStr, &metadataJSONB)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("document not found: %s", id)
		}
		return nil, err
	}

	doc := &schema.Document{
		ID:      docID,
		Content: content,
	}

	// Parse metadata
	if metadataJSONB != nil {
		var metadata map[string]any
		if err := json.Unmarshal(metadataJSONB, &metadata); err == nil {
			doc.MetaData = metadata
		}
	}

	return doc, nil
}

// DeleteDocument deletes a document by ID.
func (i *Indexer) DeleteDocument(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", i.config.TableName, i.config.IDColumn)

	result, err := i.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	return nil
}
