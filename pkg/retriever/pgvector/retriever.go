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
	"strings"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

	return r, nil
}

// Retrieve retrieves the most relevant documents for the given query.
// Implements retriever.Retriever.
func (r *Retriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	// Get common options
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK:          &r.config.DefaultTopK,
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
	var err error

	if commonOpts.Embedding != nil {
		embeddings, err := commonOpts.Embedding.EmbedStrings(ctx, []string{query})
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) > 0 && len(embeddings[0]) > 0 {
			queryVector = embeddings[0]
		}
	}

	// Build and execute the query
	documents, err := r.executeQuery(ctx, queryVector, commonOpts, retrieveOpts, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	return documents, nil
}

// executeQuery builds and executes the vector similarity search query.
func (r *Retriever) executeQuery(
	ctx context.Context,
	queryVector []float64,
	commonOpts *retriever.Options,
	retrieveOpts *RetrieveOptions,
	topK int,
) ([]*schema.Document, error) {
	var whereClauses []string
	var args []any
	argPos := 1

	// Add sub-index filtering
	if commonOpts.SubIndex != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", r.config.SubIndexesColumn, argPos))
		args = append(args, *commonOpts.SubIndex)
		argPos++
	}

	// Add filter sub-indexes (implementation-specific option)
	if len(retrieveOpts.FilterSubIndexes) > 0 {
		placeholders := make([]string, len(retrieveOpts.FilterSubIndexes))
		for i, idx := range retrieveOpts.FilterSubIndexes {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, idx)
			argPos++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s && ARRAY[%s]",
			r.config.SubIndexesColumn, strings.Join(placeholders, ", ")))
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

	// Build SELECT clause
	selectColumns := []string{
		r.config.IDColumn,
		r.config.ContentColumn,
		r.config.MetadataColumn,
	}

	if retrieveOpts.IncludeDistance {
		selectColumns = append(selectColumns,
			fmt.Sprintf("embedding %s $%d AS distance",
				r.config.DistanceFunction.operator(), argPos))
		argPos++
	}

	if retrieveOpts.IncludeVector {
		selectColumns = append(selectColumns, r.config.EmbeddingColumn)
	}

	// Build the query
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		%s
		ORDER BY %s %s $%d
		LIMIT $%d
	`,
		strings.Join(selectColumns, ", "),
		r.config.TableName,
		whereSQL,
		r.config.EmbeddingColumn,
		r.config.DistanceFunction.operator(),
		argPos,
		argPos+1,
	)

	// Add query vector as argument (used in ORDER BY)
	if queryVector != nil {
		args = append(args, vectorToString(queryVector))
	} else {
		// If no query vector, use a zero vector
		args = append(args, vectorToString(make([]float64, r.config.Dimension)))
	}
	argPos++

	// Add limit
	args = append(args, topK)

	// Execute query
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Parse results
	var documents []*schema.Document
	for rows.Next() {
		doc, err := r.scanRow(rows, retrieveOpts.IncludeDistance, retrieveOpts.IncludeVector)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Apply score threshold filtering
		if commonOpts.ScoreThreshold != nil {
			score := doc.Score()
			if score < *commonOpts.ScoreThreshold {
				continue
			}
		}

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
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

// GetPool returns the underlying connection pool.
func (r *Retriever) GetPool() *pgxpool.Pool {
	return r.pool
}

// GetConfig returns the retriever configuration.
func (r *Retriever) GetConfig() *Config {
	return r.config
}
