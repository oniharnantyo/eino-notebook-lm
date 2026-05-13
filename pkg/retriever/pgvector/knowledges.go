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
	"strings"

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

// KeywordSearch performs BM25 keyword search on the knowledges table and attaches
// KWIC (Keyword-In-Context) snippets to each document's metadata.
//
// Parameters:
//   - ctx: Context for the operation
//   - query: Space-separated keyword query string for BM25 matching
//   - keywords: Individual keywords used for KWIC snippet extraction
//   - topK: Maximum number of results to return
//
// Returns knowledge documents with KWIC snippets in metadata, ordered by BM25 score.
func (r *KnowledgesRetriever) KeywordSearch(ctx context.Context, keywords []string, topK int) ([]*schema.Document, error) {
	query := strings.Join(keywords, " ")

	docs, err := r.inner.KeywordSearch(ctx, "knowledges", query, topK, 0)
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		snippets := ExtractKeywordContexts(doc.Content, keywords, 80)
		if len(snippets) == 0 {
			snippets = []string{"[BM25 semantic match - no direct keyword occurrences found]"}
		}
		doc.MetaData["snippets"] = snippets
	}

	return docs, nil
}

// GetPool returns the underlying PostgreSQL connection pool.
func (r *KnowledgesRetriever) GetPool() *pgxpool.Pool {
	return r.inner.GetPool()
}
