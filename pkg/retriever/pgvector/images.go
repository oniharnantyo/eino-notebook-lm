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
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImagesRetriever is an adapter that provides a dedicated interface for retrieving
// image documents. It embeds the UnifiedRetriever and delegates all operations
// to the "images" table type.
//
// Deprecated: Use UnifiedRetriever directly with table type "images".
// This adapter is maintained for backward compatibility only.
// Migrate by replacing `NewImagesRetriever(pool, dim)` with `NewUnifiedRetriever(&UnifiedConfig{Pool: pool, Dimension: dim})`
// and calling `HybridRetrieve(ctx, "images", query, queryVector, topK)`.
type ImagesRetriever struct {
	inner *UnifiedRetriever
}

// NewImagesRetriever creates a new ImagesRetriever with the given configuration.
//
// The retriever is configured to query the "images" table. It uses the unified
// retriever internally for all operations.
//
// Parameters:
//   - pool: PostgreSQL connection pool with pgvector extension
//   - dimension: Dimension of the vector embeddings (must match the embedding model)
//
// Returns an error if the pool is nil or dimension is invalid.
func NewImagesRetriever(pool *pgxpool.Pool, dimension int) (*ImagesRetriever, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", dimension)
	}

	config := &UnifiedConfig{
		Pool:        pool,
		Dimension:   dimension,
		DefaultK:    60,
		DefaultTopK: 20,
	}

	inner, err := NewUnifiedRetriever(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create unified retriever: %w", err)
	}

	return &ImagesRetriever{
		inner: inner,
	}, nil
}

// Retrieve retrieves the most relevant image documents for the given query.
//
// This method implements the retriever.Retriever interface. It performs hybrid
// search combining BM25 (keyword) and vector (semantic) search using Reciprocal
// Rank Fusion (RRF) to merge results from the images table.
//
// Implements: retriever.Retriever
func (r *ImagesRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	defaultTopK := 5
	defaultScoreThreshold := 0.0
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK:           &defaultTopK,
		ScoreThreshold: &defaultScoreThreshold,
	}, opts...)

	topK := 5
	if commonOpts.TopK != nil {
		topK = *commonOpts.TopK
	}

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

	documents, err := r.inner.HybridRetrieve(ctx, "images", query, queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute hybrid search on images: %w", err)
	}

	if commonOpts.ScoreThreshold != nil {
		var filtered []*schema.Document
		for _, doc := range documents {
			if doc.Score() >= *commonOpts.ScoreThreshold {
				filtered = append(filtered, doc)
			}
		}
		return filtered, nil
	}

	return documents, nil
}

// SemanticSearch performs vector similarity search on the images table.
//
// This method searches for images that are semantically similar to the query
// vector using pgvector cosine distance.
//
// Parameters:
//   - ctx: Context for the operation
//   - queryVector: Vector embedding of the query (dimension must match config)
//   - topK: Maximum number of results to return
//
// Returns image documents ordered by similarity (closest first).
func (r *ImagesRetriever) SemanticSearch(ctx context.Context, queryVector []float64, topK int) ([]*schema.Document, error) {
	return r.inner.SemanticSearch(ctx, "images", queryVector, topK)
}

// KeywordSearch performs BM25 keyword search on the images table.
//
// This method searches for images using keyword matching with BM25 scoring
// through the pg_textsearch extension.
//
// Parameters:
//   - ctx: Context for the operation
//   - query: Keyword query string
//   - topK: Maximum number of results to return
//
// Returns image documents ordered by BM25 score (highest first).
func (r *ImagesRetriever) KeywordSearch(ctx context.Context, query string, topK int) ([]*schema.Document, error) {
	return r.inner.KeywordSearch(ctx, "images", query, topK)
}

// GetPool returns the underlying PostgreSQL connection pool.
func (r *ImagesRetriever) GetPool() *pgxpool.Pool {
	return r.inner.GetPool()
}
