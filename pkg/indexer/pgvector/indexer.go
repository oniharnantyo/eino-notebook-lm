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
	"errors"
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

	// Create extension if auto-create is enabled
	if config.AutoCreateExtension {
		if err := idx.CreateExtension(ctx); err != nil {
			return nil, fmt.Errorf("failed to create extension: %w", err)
		}
	}

	// Create table if auto-create is enabled
	if config.AutoCreateTable {
		// Drop existing table if DropBeforeCreate is enabled
		if config.DropBeforeCreate {
			if err := idx.DropTable(ctx); err != nil {
				return nil, fmt.Errorf("failed to drop table: %w", err)
			}
		}
		if err := idx.createTable(ctx); err != nil {
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create index if enabled
	if config.CreateIndexIfNotExists {
		// Drop existing index if DropBeforeCreate is enabled
		if config.DropBeforeCreate {
			if err := idx.DropIndex(ctx); err != nil {
				return nil, fmt.Errorf("failed to drop index: %w", err)
			}
		}
		if err := idx.createIndex(ctx); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Create reference ID index if ReferenceIDColumn is configured
	if config.ReferenceIDColumn != "" {
		// Drop existing reference ID index if DropBeforeCreate is enabled
		if config.DropBeforeCreate {
			if err := idx.DropReferenceIDIndex(ctx); err != nil {
				return nil, fmt.Errorf("failed to drop reference ID index: %w", err)
			}
		}
		if err := idx.createReferenceIDIndex(ctx); err != nil {
			return nil, fmt.Errorf("failed to create reference ID index: %w", err)
		}
	}

	return idx, nil
}

// Store stores documents and returns their assigned document IDs.
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

	// Prepare document IDs - use schema.Document ID for document_id column
	documentIDs := make([]string, len(docs))
	for j, doc := range docs {
		if doc.ID == "" {
			documentIDs[j] = uuid.New().String()
		} else {
			documentIDs[j] = doc.ID
		}
	}

	// Build the query based on options
	if storeOpts.Upsert {
		return i.storeUpsert(ctx, docs, documentIDs, embeddings, commonOpts)
	}

	if storeOpts.SkipExisting {
		return i.storeSkipExisting(ctx, docs, documentIDs, embeddings, commonOpts)
	}

	// Default: simple insert
	return i.storeInsert(ctx, docs, documentIDs, embeddings, commonOpts)
}

// storeInsert performs a simple insert operation.
func (i *Indexer) storeInsert(ctx context.Context, docs []*schema.Document, documentIDs []string, embeddings [][]float64, opts *indexer.Options) ([]string, error) {
	// Build columns and placeholders based on config
	var columns []string
	if i.config.ReferenceIDColumn != "" {
		columns = []string{i.config.DocumentIDColumn, i.config.ReferenceIDColumn, i.config.ContentColumn,
			i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn}
	} else {
		columns = []string{i.config.DocumentIDColumn, i.config.ContentColumn,
			i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn}
	}

	placeholders := make([]string, len(columns))
	for j := range columns {
		placeholders[j] = fmt.Sprintf("$%d", j+1)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
	`, i.config.TableName, joinQuoted(columns, ", "), joinQuoted(placeholders, ", "))

	// Begin transaction for atomicity
	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Log error but ignore if transaction was already closed (committed)
		}
	}()

	var execErr error
	for j, doc := range docs {
		// Extract reference_id from metadata if configured
		referenceID := extractReferenceID(doc.MetaData)

		if i.config.ReferenceIDColumn != "" && referenceID != "" {
			_, execErr = tx.Exec(ctx, query,
				documentIDs[j],
				referenceID,
				doc.Content,
				vectorToString(embeddings, j),
				metadataToJSONB(doc.MetaData),
				subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
			)
		} else if i.config.ReferenceIDColumn != "" {
			return nil, fmt.Errorf("failed to insert document %s: reference_id is required but not found in metadata", documentIDs[j])
		} else {
			_, execErr = tx.Exec(ctx, query,
				documentIDs[j],
				doc.Content,
				vectorToString(embeddings, j),
				metadataToJSONB(doc.MetaData),
				subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
			)
		}
		if execErr != nil {
			return nil, fmt.Errorf("failed to insert document %s: %w", documentIDs[j], execErr)
		}
	}

	// Commit transaction if all inserts succeed
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return documentIDs, nil
}

// storeUpsert performs an upsert operation (insert or update on conflict).
func (i *Indexer) storeUpsert(ctx context.Context, docs []*schema.Document, documentIDs []string, embeddings [][]float64, opts *indexer.Options) ([]string, error) {
	// Build columns and placeholders based on config
	var columns []string
	var updateColumns []string
	if i.config.ReferenceIDColumn != "" {
		columns = []string{i.config.DocumentIDColumn, i.config.ReferenceIDColumn, i.config.ContentColumn,
			i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn}
		updateColumns = []string{i.config.ReferenceIDColumn, i.config.ContentColumn, i.config.EmbeddingColumn,
			i.config.MetadataColumn, i.config.SubIndexesColumn, i.config.UpdatedAtColumn}
	} else {
		columns = []string{i.config.DocumentIDColumn, i.config.ContentColumn,
			i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn}
		updateColumns = []string{i.config.ContentColumn, i.config.EmbeddingColumn,
			i.config.MetadataColumn, i.config.SubIndexesColumn, i.config.UpdatedAtColumn}
	}

	placeholders := make([]string, len(columns))
	for j := range columns {
		placeholders[j] = fmt.Sprintf("$%d", j+1)
	}

	// Build SET clause for update
	var setClauses []string
	for _, col := range updateColumns {
		if col == i.config.UpdatedAtColumn {
			setClauses = append(setClauses, fmt.Sprintf("%s = NOW()", col))
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
		ON CONFLICT (%s) DO UPDATE SET
			%s
	`, i.config.TableName,
		joinQuoted(columns, ", "),
		joinQuoted(placeholders, ", "),
		i.config.DocumentIDColumn,
		joinQuoted(setClauses, ",\n\t\t\t"))

	// Begin transaction for atomicity
	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Log error but ignore if transaction was already closed (committed)
		}
	}()

	var execErr error
	for j, doc := range docs {
		// Extract reference_id from metadata if configured
		referenceID := extractReferenceID(doc.MetaData)

		if i.config.ReferenceIDColumn != "" && referenceID != "" {
			_, execErr = tx.Exec(ctx, query,
				documentIDs[j],
				referenceID,
				doc.Content,
				vectorToString(embeddings, j),
				metadataToJSONB(doc.MetaData),
				subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
			)
		} else if i.config.ReferenceIDColumn != "" {
			return nil, fmt.Errorf("failed to upsert document %s: reference_id is required but not found in metadata", documentIDs[j])
		} else {
			_, execErr = tx.Exec(ctx, query,
				documentIDs[j],
				doc.Content,
				vectorToString(embeddings, j),
				metadataToJSONB(doc.MetaData),
				subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
			)
		}
		if execErr != nil {
			return nil, fmt.Errorf("failed to upsert document %s: %w", documentIDs[j], execErr)
		}
	}

	// Commit transaction if all upserts succeed
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return documentIDs, nil
}

// storeSkipExisting inserts only documents that don't already exist.
func (i *Indexer) storeSkipExisting(ctx context.Context, docs []*schema.Document, documentIDs []string, embeddings [][]float64, opts *indexer.Options) ([]string, error) {
	// Build columns and placeholders based on config
	var columns []string
	if i.config.ReferenceIDColumn != "" {
		columns = []string{i.config.DocumentIDColumn, i.config.ReferenceIDColumn, i.config.ContentColumn,
			i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn}
	} else {
		columns = []string{i.config.DocumentIDColumn, i.config.ContentColumn,
			i.config.EmbeddingColumn, i.config.MetadataColumn, i.config.SubIndexesColumn}
	}

	placeholders := make([]string, len(columns))
	for j := range columns {
		placeholders[j] = fmt.Sprintf("$%d", j+1)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
		ON CONFLICT (%s) DO NOTHING
	`, i.config.TableName,
		joinQuoted(columns, ", "),
		joinQuoted(placeholders, ", "),
		i.config.DocumentIDColumn,
	)

	resultIds := make([]string, 0, len(documentIDs))

	// Begin transaction for atomicity
	tx, err := i.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for j, doc := range docs {
		// Extract reference_id from metadata if configured
		referenceID := extractReferenceID(doc.MetaData)

		if i.config.ReferenceIDColumn != "" && referenceID != "" {
			result, execErr := tx.Exec(ctx, query,
				documentIDs[j],
				referenceID,
				doc.Content,
				vectorToString(embeddings, j),
				metadataToJSONB(doc.MetaData),
				subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
			)
			if execErr != nil {
				return nil, fmt.Errorf("failed to insert document %s: %w", documentIDs[j], execErr)
			}

			// Only add ID if the row was actually inserted
			if result.RowsAffected() > 0 {
				resultIds = append(resultIds, documentIDs[j])
			}
		} else if i.config.ReferenceIDColumn != "" {
			return nil, fmt.Errorf("failed to insert document %s: reference_id is required but not found in metadata", documentIDs[j])
		} else {
			result, execErr := tx.Exec(ctx, query,
				documentIDs[j],
				doc.Content,
				vectorToString(embeddings, j),
				metadataToJSONB(doc.MetaData),
				subIndexesToArray(opts.SubIndexes, doc.SubIndexes()),
			)
			if execErr != nil {
				return nil, fmt.Errorf("failed to insert document %s: %w", documentIDs[j], execErr)
			}

			// Only add ID if the row was actually inserted
			if result.RowsAffected() > 0 {
				resultIds = append(resultIds, documentIDs[j])
			}
		}
	}

	// Commit transaction if all inserts succeed
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return resultIds, nil
}

// createTable creates the documents table if it doesn't exist.
func (i *Indexer) createTable(ctx context.Context) error {
	// Build qualified table name with schema
	qualifiedTableName := i.config.TableName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedTableName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.TableName)
	}

	// Determine metadata column type
	metadataType := "JSONB"
	if !i.config.UseJSONBForMetadata {
		metadataType = "JSON"
	}

	// Build column definitions based on config
	var columnDefs []string
	// UUID primary key with auto-generation
	columnDefs = append(columnDefs, fmt.Sprintf("%s UUID PRIMARY KEY DEFAULT gen_random_uuid()", i.config.IDColumn))
	// Document ID from schema.Document (unique constraint for conflict resolution)
	columnDefs = append(columnDefs, fmt.Sprintf("%s VARCHAR(255) NOT NULL UNIQUE", i.config.DocumentIDColumn))

	// Add ReferenceIDColumn if configured
	if i.config.ReferenceIDColumn != "" {
		columnDefs = append(columnDefs, fmt.Sprintf("%s TEXT NOT NULL", i.config.ReferenceIDColumn))
	}

	columnDefs = append(columnDefs,
		fmt.Sprintf("%s TEXT NOT NULL", i.config.ContentColumn),
		fmt.Sprintf("%s vector(%d)", i.config.EmbeddingColumn, i.config.Dimension),
		fmt.Sprintf("%s %s", i.config.MetadataColumn, metadataType),
		fmt.Sprintf("%s TEXT[]", i.config.SubIndexesColumn),
		fmt.Sprintf("%s TIMESTAMP DEFAULT NOW()", i.config.CreatedAtColumn),
		fmt.Sprintf("%s TIMESTAMP DEFAULT NOW()", i.config.UpdatedAtColumn),
	)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s
		)
	`, qualifiedTableName, joinQuoted(columnDefs, ",\n\t\t\t"))

	_, err := i.pool.Exec(ctx, query)
	return err
}

// createIndex creates an index on the embedding column if it doesn't exist.
// Uses HNSW by default, or IVFFlat if UseIVFFlat is true.
func (i *Indexer) createIndex(ctx context.Context) error {
	var query string

	// Build qualified table name with schema
	qualifiedTableName := i.config.TableName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedTableName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.TableName)
	}

	if i.config.UseIVFFlat {
		// IVFFlat index
		query = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s
			USING ivfflat (%s %s)
			WITH (lists = %d)
		`, i.config.IndexName, qualifiedTableName,
			i.config.EmbeddingColumn, i.config.DistanceFunction.indexOperator(),
			i.config.IVFLists)
	} else {
		// HNSW index with parameters
		query = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s
			USING hnsw (%s %s)
			WITH (m = %d, ef_construction = %d)
		`, i.config.IndexName, qualifiedTableName,
			i.config.EmbeddingColumn, i.config.DistanceFunction.indexOperator(),
			i.config.HNSWM, i.config.HNSWEFConstruction)
	}

	_, err := i.pool.Exec(ctx, query)
	return err
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
	// Build qualified table name with schema
	qualifiedTableName := i.config.TableName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedTableName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.TableName)
	}

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", qualifiedTableName)
	_, err := i.pool.Exec(ctx, query)
	return err
}

// DropIndex drops the embedding index.
func (i *Indexer) DropIndex(ctx context.Context) error {
	// Build qualified index name with schema
	qualifiedIndexName := i.config.IndexName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedIndexName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.IndexName)
	}

	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", qualifiedIndexName)
	_, err := i.pool.Exec(ctx, query)
	return err
}

// createReferenceIDIndex creates a btree index on the ReferenceIDColumn if configured.
func (i *Indexer) createReferenceIDIndex(ctx context.Context) error {
	if i.config.ReferenceIDColumn == "" {
		return nil // No reference ID column configured
	}

	// Build qualified table name with schema
	qualifiedTableName := i.config.TableName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedTableName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.TableName)
	}

	// Build qualified index name with schema
	qualifiedIndexName := i.config.ReferenceIDIndexName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedIndexName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.ReferenceIDIndexName)
	}

	query := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s ON %s
		USING btree (%s)
	`, qualifiedIndexName, qualifiedTableName, i.config.ReferenceIDColumn)

	_, err := i.pool.Exec(ctx, query)
	return err
}

// DropReferenceIDIndex drops the reference ID index.
func (i *Indexer) DropReferenceIDIndex(ctx context.Context) error {
	if i.config.ReferenceIDColumn == "" {
		return nil // No reference ID column configured
	}

	// Build qualified index name with schema
	qualifiedIndexName := i.config.ReferenceIDIndexName
	if i.config.TableSchema != "" && i.config.TableSchema != "public" {
		qualifiedIndexName = fmt.Sprintf("%s.%s", i.config.TableSchema, i.config.ReferenceIDIndexName)
	}

	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", qualifiedIndexName)
	_, err := i.pool.Exec(ctx, query)
	return err
}

// TableExists checks if the documents table exists.
func (i *Indexer) TableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = $1
			AND table_name = $2
		)
	`

	schema := i.config.TableSchema
	if schema == "" {
		schema = "public"
	}

	var exists bool
	err := i.pool.QueryRow(ctx, query, schema, i.config.TableName).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// DocumentExists checks if a document with the given document ID exists.
func (i *Indexer) DocumentExists(ctx context.Context, id string) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", i.config.TableName, i.config.DocumentIDColumn)

	var exists bool
	err := i.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetDocument retrieves a document by document ID.
func (i *Indexer) GetDocument(ctx context.Context, id string) (*schema.Document, error) {
	query := fmt.Sprintf(`
		SELECT %s, %s, %s, %s
		FROM %s
		WHERE %s = $1
	`, i.config.DocumentIDColumn, i.config.ContentColumn, i.config.EmbeddingColumn, i.config.MetadataColumn,
		i.config.TableName, i.config.DocumentIDColumn)

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

// DeleteDocument deletes a document by document ID.
func (i *Indexer) DeleteDocument(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", i.config.TableName, i.config.DocumentIDColumn)

	result, err := i.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	return nil
}
