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
	"golang.org/x/sync/errgroup"
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
		BM25Index:  "public.knowledges_bm25_idx",
		JoinClause: "",
	}
	r.tables["sentences"] = TableConfig{
		Name:       "sentences",
		BM25Index:  "public.sentences_bm25_idx",
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

	query := fmt.Sprintf(`
		SELECT id, content, metadata, embedding %s $1 AS distance
		FROM %s
		ORDER BY embedding %s $1
		LIMIT $2
	`,
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
) ([]*schema.Document, error) {
	tableConfig, err := r.GetTableConfig(tableType)
	if err != nil {
		return nil, err
	}

	escapedQuery := strings.ReplaceAll(query, "'", "''")

	bm25Query := fmt.Sprintf(`
		SELECT id, content, metadata, content <@> '%s' AS bm25_score
		FROM %s
		WHERE content <@> to_bm25query('%s', '%s') < 0
		ORDER BY content <@> '%s'
		LIMIT $1
	`,
		escapedQuery,
		tableConfig.Name,
		escapedQuery,
		tableConfig.BM25Index,
		escapedQuery,
	)

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

// HybridRetrieve performs hybrid retrieval using RRF fusion of semantic and keyword search.
func (r *UnifiedRetriever) HybridRetrieve(
	ctx context.Context,
	tableType string,
	query string,
	queryVector []float64,
	topK int,
) ([]*schema.Document, error) {
	tableConfig, err := r.GetTableConfig(tableType)
	if err != nil {
		return nil, err
	}

	k := r.config.DefaultK
	if k <= 0 {
		k = 60
	}

	candidatesK := r.config.DefaultTopK
	if candidatesK <= 0 {
		candidatesK = 20
	}

	g, egCtx := errgroup.WithContext(ctx)

	vectorRanked := make(chan []RankedDocument, 1)
	bm25Ranked := make(chan []RankedDocument, 1)

	g.Go(func() error {
		defer close(vectorRanked)
		results, err := r.semanticSearchRanked(egCtx, tableConfig, queryVector, candidatesK)
		if err != nil {
			return fmt.Errorf("vector search failed: %w", err)
		}
		vectorRanked <- results
		return nil
	})

	g.Go(func() error {
		defer close(bm25Ranked)
		results, err := r.keywordSearchRanked(egCtx, tableConfig, query, candidatesK)
		if err != nil {
			return fmt.Errorf("keyword search failed: %w", err)
		}
		bm25Ranked <- results
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	vResults := <-vectorRanked
	bResults := <-bm25Ranked

	rrfScores := MergeByRRF(k, vResults, bResults)
	rankedResults := SortByScore(rrfScores)
	topNResults := TopN(rankedResults, topK)

	documents, err := r.rankedListToDocuments(ctx, tableConfig, topNResults)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ranked results to documents: %w", err)
	}

	slog.Info("unified_hybrid_retrieve",
		"table_type", tableType,
		"table", tableConfig.Name,
		"rrf_k", k,
		"candidates_k", candidatesK,
		"top_k", topK,
		"results", len(documents),
	)

	return documents, nil
}

// semanticSearchRanked performs semantic search and returns ranked documents.
func (r *UnifiedRetriever) semanticSearchRanked(
	ctx context.Context,
	tableConfig TableConfig,
	queryVector []float64,
	topK int,
) ([]RankedDocument, error) {
	if len(queryVector) != r.config.Dimension {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d",
			r.config.Dimension, len(queryVector))
	}

	query := fmt.Sprintf(`
		SELECT id
		FROM %s
		ORDER BY embedding %s $1
		LIMIT $2
	`,
		tableConfig.Name,
		DistanceCosine.operator(),
	)

	vecStr := vectorToString(queryVector)
	rows, err := r.pool.Query(ctx, query, vecStr, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search query failed: %w", err)
	}
	defer rows.Close()

	var results []RankedDocument
	rank := 1
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan vector row: %w", err)
		}
		results = append(results, RankedDocument{
			ID:   id,
			Rank: rank,
		})
		rank++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vector row iteration error: %w", err)
	}

	return results, nil
}

// keywordSearchRanked performs keyword search and returns ranked documents.
func (r *UnifiedRetriever) keywordSearchRanked(
	ctx context.Context,
	tableConfig TableConfig,
	query string,
	topK int,
) ([]RankedDocument, error) {
	escapedQuery := strings.ReplaceAll(query, "'", "''")

	bm25Query := fmt.Sprintf(`
		SELECT id
		FROM %s
		WHERE content <@> to_bm25query('%s', '%s') < 0
		ORDER BY content <@> '%s'
		LIMIT $1
	`,
		tableConfig.Name,
		escapedQuery,
		tableConfig.BM25Index,
		escapedQuery,
	)

	rows, err := r.pool.Query(ctx, bm25Query, topK)
	if err != nil {
		return nil, fmt.Errorf("keyword search query failed: %w", err)
	}
	defer rows.Close()

	var results []RankedDocument
	rank := 1
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan keyword row: %w", err)
		}
		results = append(results, RankedDocument{
			ID:   id,
			Rank: rank,
		})
		rank++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("keyword row iteration error: %w", err)
	}

	return results, nil
}

// rankedListToDocuments converts ranked document IDs to full schema.Document objects.
func (r *UnifiedRetriever) rankedListToDocuments(
	ctx context.Context,
	tableConfig TableConfig,
	ranked []RankedDocumentWithScore,
) ([]*schema.Document, error) {
	if len(ranked) == 0 {
		return []*schema.Document{}, nil
	}

	ids := make([]string, len(ranked))
	for i, r := range ranked {
		ids[i] = r.ID
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, content, metadata
		FROM %s
		WHERE id IN (%s)
	`,
		tableConfig.Name,
		strings.Join(placeholders, ", "),
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	defer rows.Close()

	scoreMap := make(map[string]float64)
	for _, r := range ranked {
		scoreMap[r.ID] = r.Score
	}

	docMap := make(map[string]*schema.Document)
	for rows.Next() {
		var id, content string
		var metadataJSONB []byte

		if err := rows.Scan(&id, &content, &metadataJSONB); err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
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

		if score, ok := scoreMap[id]; ok {
			doc.WithScore(score)
		}

		docMap[id] = doc
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document row iteration error: %w", err)
	}

	documents := make([]*schema.Document, 0, len(ranked))
	for _, r := range ranked {
		if doc, ok := docMap[r.ID]; ok {
			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// GetPool returns the underlying connection pool.
func (r *UnifiedRetriever) GetPool() *pgxpool.Pool {
	return r.pool
}
