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

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// Retriever is an implementation of retriever.Retriever using PostgreSQL with pgvector.
type Retriever struct {
	config *Config
	pool   *pgxpool.Pool
}

// NewRetriever creates a new pgvector retriever.
func NewRetriever(ctx context.Context, config *Config) (*Retriever, error) {
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

	r := &Retriever{
		config: config,
		pool:   config.Pool,
	}

	// Create pg_textsearch extension if auto-create is enabled
	if config.AutoCreateBM25Extension {
		if err := r.CreateBM25Extension(ctx); err != nil {
			return nil, fmt.Errorf("failed to create pg_textsearch extension: %w", err)
		}
	}

	// Create BM25 index if auto-create is enabled
	if config.AutoCreateBM25Index {
		// Drop existing index if DropBeforeCreate is enabled
		if config.DropBeforeCreate {
			if err := r.DropBM25Index(ctx); err != nil {
				return nil, fmt.Errorf("failed to drop BM25 index: %w", err)
			}
		}
		if err := r.createBM25Index(ctx); err != nil {
			return nil, fmt.Errorf("failed to create BM25 index: %w", err)
		}
	}

	return r, nil
}

// Retrieve retrieves the most relevant documents for the given query.
// Implements retriever.Retriever using hybrid search (BM25 + vector with RRF).
func (r *Retriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	// Get common options
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK:           &r.config.DefaultTopK,
		ScoreThreshold: &r.config.DefaultScoreThreshold,
	}, opts...)

	retrieveOpts := getRetrieveOptions(opts...)

	// Get topK
	topK := 5
	if commonOpts.TopK != nil {
		topK = *commonOpts.TopK
	}

	// Generate embedding for the query if an embedder is provided
	var queryVector []float64

	if commonOpts.Embedding != nil {
		embeddings, err := commonOpts.Embedding.EmbedStrings(ctx, []string{query})
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) > 0 && len(embeddings[0]) > 0 {
			queryVector = embeddings[0]
		}
	}

	// Execute hybrid search in parallel
	documents, err := r.executeHybridSearch(ctx, query, queryVector, commonOpts, retrieveOpts, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute hybrid search: %w", err)
	}

	return documents, nil
}

// executeHybridSearch runs BM25 and vector searches in parallel and merges results using RRF.
func (r *Retriever) executeHybridSearch(
	ctx context.Context,
	query string,
	queryVector []float64,
	commonOpts *retriever.Options,
	retrieveOpts *RetrieveOptions,
	topK int,
) ([]*schema.Document, error) {
	// Use errgroup for parallel execution with context cancellation
	g, egCtx := errgroup.WithContext(ctx)

	// Channels to collect results
	vectorResults := make(chan []RankedDocument, 1)
	bm25Results := make(chan []RankedDocument, 1)

	// Run vector search in parallel
	g.Go(func() error {
		defer close(vectorResults)
		results, err := r.executeVectorSearch(egCtx, queryVector, retrieveOpts)
		if err != nil {
			return fmt.Errorf("vector search failed: %w", err)
		}
		vectorResults <- results
		return nil
	})

	// Run BM25 search in parallel
	g.Go(func() error {
		defer close(bm25Results)
		results, err := r.executeBM25SearchRanked(egCtx, query, retrieveOpts)
		if err != nil {
			return fmt.Errorf("bm25 search failed: %w", err)
		}
		bm25Results <- results
		return nil
	})

	// Wait for both searches to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Collect results
	vResults := <-vectorResults
	bResults := <-bm25Results

	// Merge using RRF
	rrfScores := MergeByRRF(retrieveOpts.RRFK, vResults, bResults)

	// Sort by RRF score and get top N
	rankedResults := SortByScore(rrfScores)
	topNResults := TopN(rankedResults, topK)

	// Convert ranked results to schema.Documents
	documents, err := r.rankedListToDocuments(ctx, topNResults, retrieveOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ranked results to documents: %w", err)
	}

	// Apply score threshold filtering
	var filtered []*schema.Document
	if commonOpts.ScoreThreshold != nil {
		for _, doc := range documents {
			if doc.Score() >= *commonOpts.ScoreThreshold {
				filtered = append(filtered, doc)
			}
		}
	} else {
		filtered = documents
	}

	// Log final merged/RRF results
	slog.Info("rrf_merged_results",
		"top_k", topK,
		"rrf_k", retrieveOpts.RRFK,
		"threshold", commonOpts.ScoreThreshold,
		"before_threshold", len(documents),
		"after_threshold", len(filtered),
		"results", formatDocumentResults(filtered),
	)

	// Apply reranker if configured and not skipped
	if r.config.BM25IndexName != "" && !retrieveOpts.SkipRerank {
		// Reranker would be applied here if one was configured
		// For now, we just return the RRF results
	}

	return filtered, nil
}

// executeVectorSearch performs vector similarity search and returns ranked documents.
func (r *Retriever) executeVectorSearch(
	ctx context.Context,
	queryVector []float64,
	retrieveOpts *RetrieveOptions,
) ([]RankedDocument, error) {
	var whereClauses []string
	var args []any
	argPos := 1

	// Add reference ID filtering
	if r.config.ReferenceIDColumn != "" && len(retrieveOpts.FilterReferenceIDs) > 0 {
		placeholders := make([]string, len(retrieveOpts.FilterReferenceIDs))
		for i, id := range retrieveOpts.FilterReferenceIDs {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, id)
			argPos++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)",
			r.config.ReferenceIDColumn, strings.Join(placeholders, ", ")))
	}

	// Add source_type filtering
	if len(retrieveOpts.FilterSourceTypes) > 0 {
		placeholders := make([]string, len(retrieveOpts.FilterSourceTypes))
		for i, sourceType := range retrieveOpts.FilterSourceTypes {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, sourceType)
			argPos++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("metadata->>'source_type' IN (%s)",
			strings.Join(placeholders, ", ")))
	}

	// Add custom WHERE clause
	if retrieveOpts.WhereClause != "" {
		whereClauses = append(whereClauses, retrieveOpts.WhereClause)
	}

	// Build WHERE clause
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build the query
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		%s
		ORDER BY %s %s $%d
		LIMIT $%d
	`,
		r.config.IDColumn,
		r.config.TableName,
		whereSQL,
		r.config.EmbeddingColumn,
		r.config.DistanceFunction.operator(),
		argPos,
		argPos+1,
	)

	// Add query vector
	vecStr := ""
	if queryVector != nil {
		vecStr = vectorToString(queryVector)
	} else {
		vecStr = vectorToString(make([]float64, r.config.Dimension))
	}
	args = append(args, vecStr)

	// Add limit
	args = append(args, retrieveOpts.VectorCandidates)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("vector query failed: %w", err)
	}
	defer rows.Close()

	// Parse results
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

	// Log vector search results
	slog.Info("vector_search_results",
		"count", len(results),
		"results", formatRankedResults(results),
	)

	return results, nil
}

// executeBM25SearchRanked performs BM25 search and returns ranked documents.
func (r *Retriever) executeBM25SearchRanked(
	ctx context.Context,
	query string,
	retrieveOpts *RetrieveOptions,
) ([]RankedDocument, error) {
	results, err := r.executeBM25Search(ctx, query, retrieveOpts, retrieveOpts.BM25Candidates)
	if err != nil {
		return nil, err
	}

	// Convert BM25Result to RankedDocument (BM25 returns sorted results)
	ranked := make([]RankedDocument, len(results))
	for i, result := range results {
		ranked[i] = RankedDocument{
			ID:   result.ID,
			Rank: i + 1, // 1-indexed rank
		}
	}

	// Log BM25 search results
	slog.Info("bm25_search_results",
		"count", len(ranked),
		"results", formatRankedResults(ranked),
	)

	return ranked, nil
}

// rankedListToDocuments converts ranked document IDs to full schema.Document objects.
func (r *Retriever) rankedListToDocuments(
	ctx context.Context,
	ranked []RankedDocumentWithScore,
	retrieveOpts *RetrieveOptions,
) ([]*schema.Document, error) {
	if len(ranked) == 0 {
		return []*schema.Document{}, nil
	}

	// Extract document IDs
	ids := make([]string, len(ranked))
	for i, r := range ranked {
		ids[i] = r.ID
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Query to fetch full documents
	query := fmt.Sprintf(`
		SELECT %s, %s, %s
		FROM %s
		WHERE %s IN (%s)
	`,
		r.config.IDColumn,
		r.config.ContentColumn,
		r.config.MetadataColumn,
		r.config.TableName,
		r.config.IDColumn,
		strings.Join(placeholders, ", "),
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}
	defer rows.Close()

	// Map ID to score
	scoreMap := make(map[string]float64)
	for _, r := range ranked {
		scoreMap[r.ID] = r.Score
	}

	// Parse results
	docMap := make(map[string]*schema.Document)
	for rows.Next() {
		var id, content string
		var metadataJSONB []byte

		if err := rows.Scan(&id, &content, &metadataJSONB); err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		doc := &schema.Document{
			ID:      id,
			Content: content,
			MetaData: make(map[string]any),
		}

		// Parse metadata
		if metadataJSONB != nil {
			var metadata map[string]any
			if err := json.Unmarshal(metadataJSONB, &metadata); err == nil {
				for k, v := range metadata {
					doc.MetaData[k] = v
				}
			}
		}

		// Set RRF score
		if score, ok := scoreMap[id]; ok {
			doc.WithScore(score)
		}

		docMap[id] = doc
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document row iteration error: %w", err)
	}

	// Return documents in the ranked order
	documents := make([]*schema.Document, 0, len(ranked))
	for _, r := range ranked {
		if doc, ok := docMap[r.ID]; ok {
			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// scanRow scans a single row from the query result.
func (r *Retriever) scanRow(rows pgx.Rows, includeDistance, includeVector bool) (*schema.Document, error) {
	var id, content string
	var metadataJSONB []byte
	var distance *float64
	var vectorBytes []byte

	// Build scan targets based on what we're selecting
	scanTargets := []any{&id, &content, &metadataJSONB}
	if includeDistance {
		scanTargets = append(scanTargets, &distance)
	}
	if includeVector {
		scanTargets = append(scanTargets, &vectorBytes)
	}

	if err := rows.Scan(scanTargets...); err != nil {
		return nil, err
	}

	doc := &schema.Document{
		ID:      id,
		Content: content,
	}

	// Initialize metadata
	doc.MetaData = make(map[string]any)

	// Parse metadata from JSONB
	if metadataJSONB != nil {
		var metadata map[string]any
		if err := json.Unmarshal(metadataJSONB, &metadata); err == nil {
			// Merge metadata
			for k, v := range metadata {
				doc.MetaData[k] = v
			}
		}
	}

	// Add distance/score to metadata
	if distance != nil {
		score := *distance
		// For cosine distance, convert to similarity (1 - distance)
		// For other distances, use the raw value
		if r.config.DistanceFunction == DistanceCosine {
			score = 1 - score
		}
		doc.WithScore(score)
	}

	// Add vector to metadata if requested
	if includeVector && vectorBytes != nil {
		doc.MetaData["_vector"] = string(vectorBytes)
	}

	return doc, nil
}

// vectorToString converts a vector to PostgreSQL vector format.
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

// formatRankedResults formats ranked documents for logging.
func formatRankedResults(results []RankedDocument) string {
	if len(results) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.Grow(len(results)*20) // Pre-allocate approximate capacity

	builder.WriteString("[")
	for i, r := range results {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("{id:%s,rank:%d}", r.ID, r.Rank))
	}
	builder.WriteString("]")

	return builder.String()
}

// formatDocumentResults formats documents for logging.
func formatDocumentResults(docs []*schema.Document) string {
	if len(docs) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.Grow(len(docs)*30) // Pre-allocate approximate capacity

	builder.WriteString("[")
	for i, doc := range docs {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("{id:%s,score:%.4f}", doc.ID, doc.Score()))
	}
	builder.WriteString("]")

	return builder.String()
}

// GetPool returns the underlying connection pool.
func (r *Retriever) GetPool() *pgxpool.Pool {
	return r.pool
}

// GetConfig returns the retriever configuration.
func (r *Retriever) GetConfig() *Config {
	return r.config
}

// CreateBM25Extension creates the pg_textsearch extension in the database.
func (r *Retriever) CreateBM25Extension(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS pg_textsearch")
	return err
}

// createBM25Index creates a BM25 index on the content column using pg_textsearch.
func (r *Retriever) createBM25Index(ctx context.Context) error {
	// PostgreSQL doesn't allow schema-qualified index names in CREATE INDEX
	// The index inherits the schema from the table, so we strip any schema prefix.
	indexName := r.config.BM25IndexName
	if idx := strings.LastIndex(indexName, "."); idx != -1 {
		indexName = indexName[idx+1:]
	}

	query := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s ON %s
		USING bm25(%s) WITH (text_config='%s')
	`, indexName, r.config.TableName,
		r.config.ContentColumn, r.config.BM25TextConfig)

	_, err := r.pool.Exec(ctx, query)
	return err
}

// DropBM25Index drops the BM25 index if it exists.
func (r *Retriever) DropBM25Index(ctx context.Context) error {
	// Strip schema prefix for DROP INDEX as well
	indexName := r.config.BM25IndexName
	if idx := strings.LastIndex(indexName, "."); idx != -1 {
		indexName = indexName[idx+1:]
	}

	query := fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
	_, err := r.pool.Exec(ctx, query)
	return err
}

// BM25IndexExists checks if the BM25 index exists.
func (r *Retriever) BM25IndexExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM pg_indexes
			WHERE indexname = $1
		)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, r.config.BM25IndexName).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
