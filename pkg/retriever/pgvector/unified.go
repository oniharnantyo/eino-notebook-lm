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
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TableConfig holds configuration for a specific table type.
type TableConfig struct {
	// Name is the table name in the database (e.g., "knowledges", "sentences", "images")
	Name string
	// BM25Index is the name of the BM25 index for this table (e.g., "knowledges_bm25_idx")
	BM25Index string
	// JoinClause is an optional SQL JOIN clause for queries requiring joins (e.g., sentences -> knowledges)
	JoinClause string
}

// UnifiedConfig holds configuration for the unified retriever.
type UnifiedConfig struct {
	// Pool is the PostgreSQL connection pool.
	Pool *pgxpool.Pool
	// Dimension is the dimension of the vector embeddings.
	Dimension int
	// DefaultK is the default RRF constant (default 60)
	DefaultK int
	// DefaultTopK is the default number of candidates to fetch from each method (default 20)
	DefaultTopK int
}

// UnifiedRetriever is a unified retriever that can query multiple table types.
type UnifiedRetriever struct {
	config *UnifiedConfig
	pool   *pgxpool.Pool
	tables map[string]TableConfig
}

// NewUnifiedRetriever creates a new unified retriever with default table configurations.
func NewUnifiedRetriever(config *UnifiedConfig) (*UnifiedRetriever, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.Pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", config.Dimension)
	}

	r := &UnifiedRetriever{
		config: config,
		pool:   config.Pool,
		tables: make(map[string]TableConfig),
	}

	r.registerDefaultTables()

	return r, nil
}

// registerDefaultTables registers the default table configurations.
func (r *UnifiedRetriever) registerDefaultTables() {
	r.tables["knowledges"] = TableConfig{
		Name:       "knowledges",
		BM25Index:  "public.idx_knowledges_content_bm25",
		JoinClause: "",
	}
	r.tables["sentences"] = TableConfig{
		Name:       "sentences",
		BM25Index:  "public.idx_sentences_content_bm25",
		JoinClause: "",
	}
	r.tables["images"] = TableConfig{
		Name:       "images",
		BM25Index:  "public.images_bm25_idx",
		JoinClause: "",
	}
}

// RegisterTable registers a custom table configuration.
func (r *UnifiedRetriever) RegisterTable(tableType string, config TableConfig) error {
	if config.Name == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	r.tables[tableType] = config
	return nil
}

// GetTableConfig returns the configuration for a table type.
func (r *UnifiedRetriever) GetTableConfig(tableType string) (TableConfig, error) {
	config, ok := r.tables[tableType]
	if !ok {
		return TableConfig{}, fmt.Errorf("unknown table type: %s", tableType)
	}
	return config, nil
}

// SemanticSearch performs vector similarity search on the specified table.
func (r *UnifiedRetriever) SemanticSearch(
	ctx context.Context,
	tableType string,
	queryVector []float64,
	topK int,
) ([]*schema.Document, error) {
	tableConfig, err := r.GetTableConfig(tableType)
	if err != nil {
		return nil, err
	}

	if len(queryVector) != r.config.Dimension {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d",
			r.config.Dimension, len(queryVector))
	}

	idCol := "id"
	if tableType == "sentences" {
		idCol = "knowledge_id"
	}

	query := fmt.Sprintf(`
		SELECT %s AS id, content, metadata, embedding %s $1 AS distance
		FROM %s
		ORDER BY embedding %s $1
		LIMIT $2
	`,
		idCol,
		DistanceCosine.operator(),
		tableConfig.Name,
		DistanceCosine.operator(),
	)

	vecStr := vectorToString(queryVector)
	rows, err := r.pool.Query(ctx, query, vecStr, topK)
	if err != nil {
		return nil, fmt.Errorf("semantic search query failed: %w", err)
	}
	defer rows.Close()

	var documents []*schema.Document
	for rows.Next() {
		var id, content string
		var metadataJSONB []byte
		var distance float64

		if err := rows.Scan(&id, &content, &metadataJSONB, &distance); err != nil {
			return nil, fmt.Errorf("failed to scan semantic search row: %w", err)
		}

		doc := &schema.Document{
			ID:       id,
			Content:  content,
			MetaData: make(map[string]any),
		}

		if metadataJSONB != nil {
			var metadata map[string]any
			if err := json.Unmarshal(metadataJSONB, &metadata); err == nil {
				for k, v := range metadata {
					doc.MetaData[k] = v
				}
			}
		}

		doc.WithScore(1 - distance)
		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("semantic search row iteration error: %w", err)
	}

	slog.Info("unified_semantic_search",
		"table_type", tableType,
		"table", tableConfig.Name,
		"top_k", topK,
		"results", len(documents),
	)

	return documents, nil
}

// KeywordSearch performs BM25 keyword search on the specified table.
func (r *UnifiedRetriever) KeywordSearch(
	ctx context.Context,
	tableType string,
	query string,
	topK int,
	scoreThreshold float64,
) ([]*schema.Document, error) {
	tableConfig, err := r.GetTableConfig(tableType)
	if err != nil {
		return nil, err
	}

	escapedQuery := strings.ReplaceAll(query, "'", "''")

	var bm25Query string
	if scoreThreshold < 0 {
		// Use WHERE clause with explicit to_bm25query() for threshold filtering
		bm25Query = fmt.Sprintf(`
			SELECT id, content, metadata, content <@> '%s' AS bm25_score
			FROM %s
			WHERE content <@> to_bm25query('%s', '%s') < %.2f
			ORDER BY content <@> '%s'
			LIMIT $1
		`,
			escapedQuery,
			tableConfig.Name,
			escapedQuery,
			tableConfig.BM25Index,
			scoreThreshold,
			escapedQuery,
		)
	} else {
		// No threshold - use implicit query syntax for better performance
		bm25Query = fmt.Sprintf(`
			SELECT id, content, metadata, content <@> '%s' AS bm25_score
			FROM %s
			ORDER BY content <@> '%s'
			LIMIT $1
		`,
			escapedQuery,
			tableConfig.Name,
			escapedQuery,
		)
	}

	rows, err := r.pool.Query(ctx, bm25Query, topK)
	if err != nil {
		return nil, fmt.Errorf("keyword search query failed: %w", err)
	}
	defer rows.Close()

	var documents []*schema.Document
	for rows.Next() {
		var id, content string
		var metadataJSONB []byte
		var bm25Score float64

		if err := rows.Scan(&id, &content, &metadataJSONB, &bm25Score); err != nil {
			return nil, fmt.Errorf("failed to scan keyword search row: %w", err)
		}

		doc := &schema.Document{
			ID:       id,
			Content:  content,
			MetaData: make(map[string]any),
		}

		if metadataJSONB != nil {
			var metadata map[string]any
			if err := json.Unmarshal(metadataJSONB, &metadata); err == nil {
				for k, v := range metadata {
					doc.MetaData[k] = v
				}
			}
		}

		doc.WithScore(-bm25Score)
		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("keyword search row iteration error: %w", err)
	}

	slog.Info("unified_keyword_search",
		"table_type", tableType,
		"table", tableConfig.Name,
		"bm25_index", tableConfig.BM25Index,
		"top_k", topK,
		"results", len(documents),
	)

	return documents, nil
}

// GetPool returns the underlying connection pool.
func (r *UnifiedRetriever) GetPool() *pgxpool.Pool {
	return r.pool
}

// vectorToString converts a float64 slice to a string representation for PostgreSQL.
func vectorToString(vector []float64) string {
	if len(vector) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.Grow(len(vector)*8 + 2) // Pre-allocate approximate capacity

	builder.WriteString("[")
	for i, v := range vector {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf("%g", v))
	}
	builder.WriteString("]")

	return builder.String()
}
