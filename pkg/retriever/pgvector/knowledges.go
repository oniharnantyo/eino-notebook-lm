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

// KnowledgesRetriever is a deprecated adapter for retrieving knowledge documents.
//
// Deprecated: Use UnifiedRetriever directly instead. This adapter exists for backward
// compatibility only. New code should use UnifiedRetriever with table type "knowledges".
//
// Migration path:
//   - Old: NewKnowledgesRetriever(pool, dimension)
//   - New: NewUnifiedRetriever(&UnifiedConfig{Pool: pool, Dimension: dimension})
//     Then use: HybridRetrieve(ctx, "knowledges", query, queryVector, topK)
//
// This adapter maintains backward compatibility for existing code, while internally
// using the unified retrieval implementation.
//
// This adapter will be removed in a future release.
type KnowledgesRetriever struct {
	inner *UnifiedRetriever
}

// NewKnowledgesRetriever creates a new KnowledgesRetriever with the given configuration.
//
// Deprecated: Use NewUnifiedRetriever instead. This constructor exists for backward
// compatibility only. See KnowledgesRetriever type documentation for migration path.
//
// The retriever is configured to query the "knowledges" table. It uses the unified
// retriever internally for all operations.
//
// Parameters:
//   - pool: PostgreSQL connection pool with pgvector extension
//   - dimension: Dimension of the vector embeddings (must match the embedding model)
//
// Returns an error if the pool is nil or dimension is invalid.
func NewKnowledgesRetriever(pool *pgxpool.Pool, dimension int) (*KnowledgesRetriever, error) {
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

	return &KnowledgesRetriever{
		inner: inner,
	}, nil
}

// Retrieve retrieves the most relevant knowledge documents for the given query.
//
// This method implements the retriever.Retriever interface. It performs hybrid
// search combining BM25 (keyword) and vector (semantic) search using Reciprocal
// Rank Fusion (RRF) to merge results from the knowledges table.
//
// Implements: retriever.Retriever
func (r *KnowledgesRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
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

	documents, err := r.inner.HybridRetrieve(ctx, "knowledges", query, queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to execute hybrid search on knowledges: %w", err)
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

// SemanticSearch performs vector similarity search on the knowledges table.
//
// This method searches for knowledge documents that are semantically similar to the query
// vector using pgvector cosine distance.
//
// Parameters:
//   - ctx: Context for the operation
//   - queryVector: Vector embedding of the query (dimension must match config)
//   - topK: Maximum number of results to return
//
// Returns knowledge documents ordered by similarity (closest first).
func (r *KnowledgesRetriever) SemanticSearch(ctx context.Context, queryVector []float64, topK int) ([]*schema.Document, error) {
	return r.inner.SemanticSearch(ctx, "knowledges", queryVector, topK)
}

// KeywordSearch performs BM25 keyword search on the knowledges table.
//
// This method searches for knowledge documents using keyword matching with BM25 scoring
// through the pg_textsearch extension.
//
// Parameters:
//   - ctx: Context for the operation
//   - query: Keyword query string
//   - topK: Maximum number of results to return
//
// Returns knowledge documents ordered by BM25 score (highest first).
func (r *KnowledgesRetriever) KeywordSearch(ctx context.Context, query string, topK int) ([]*schema.Document, error) {
	return r.inner.KeywordSearch(ctx, "knowledges", query, topK)
}

// GetPool returns the underlying PostgreSQL connection pool.
func (r *KnowledgesRetriever) GetPool() *pgxpool.Pool {
	return r.inner.GetPool()
}
