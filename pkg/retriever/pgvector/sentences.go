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

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SentencesRetriever is a deprecated adapter for retrieving sentence documents.
//
// Deprecated: Use UnifiedRetriever directly instead. This adapter exists for backward
// compatibility only. New code should use UnifiedRetriever with table type "sentences".
//
// Migration path:
//   - Old: NewSentencesRetriever(ctx, pool, dimension)
//   - New: NewUnifiedRetriever(&UnifiedConfig{Pool: pool, Dimension: dimension})
//     Then use: HybridRetrieve(ctx, "sentences", query, queryVector, topK)
//
// This adapter maintains backward compatibility for existing code, while internally
// using the unified retrieval implementation. Sentences may JOIN with the knowledges
// table to fetch parent metadata, which is handled through the JoinClause in the
// table configuration.
//
// This adapter will be removed in a future release.
type SentencesRetriever struct {
	inner    *UnifiedRetriever
	embedder embedding.Embedder
}

// NewSentencesRetriever creates a new SentencesRetriever with the given configuration.
//
// Deprecated: Use NewUnifiedRetriever instead. This constructor exists for backward
// compatibility only. See SentencesRetriever type documentation for migration path.
//
// The retriever is configured to query the "sentences" table. It uses the unified
// retriever internally for all operations.
//
// Parameters:
//   - pool: PostgreSQL connection pool with pgvector extension
//   - dimension: Dimension of the vector embeddings (must match the embedding model)
//
// Returns an error if the pool is nil or dimension is invalid.
func NewSentencesRetriever(pool *pgxpool.Pool, dimension int, embedder embedding.Embedder) (*SentencesRetriever, error) {
	if pool == nil {
		return nil, fmt.Errorf("connection pool cannot be nil")
	}
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", dimension)
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
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

	return &SentencesRetriever{
		inner:    inner,
		embedder: embedder,
	}, nil
}

// SemanticSearch performs semantic similarity search on the sentences table.
//
// This method embeds the query string using the configured embedder, then performs
// vector similarity search using pgvector cosine distance.
//
// Parameters:
//   - ctx: Context for the operation
//   - query: Natural language query string
//   - topK: Maximum number of results to return
//
// Returns sentence documents ordered by similarity (closest first).
func (r *SentencesRetriever) SemanticSearch(ctx context.Context, query string, topK int) ([]*schema.Document, error) {
	vecs, err := r.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, fmt.Errorf("embedder returned empty vector for query")
	}

	docs, err := r.inner.SemanticSearch(ctx, "sentences", vecs[0], topK)
	if err != nil {
		return nil, err
	}

	for _, doc := range docs {
		if len(doc.Content) > 200 {
			doc.Content = doc.Content[:200] + "..."
		}
	}

	return docs, nil
}

// KeywordSearch performs BM25 keyword search on the sentences table.
//
// This method searches for sentences using keyword matching with BM25 scoring
// through the pg_textsearch extension.
//
// Parameters:
//   - ctx: Context for the operation
//   - query: Keyword query string
//   - topK: Maximum number of results to return
//
// Returns sentence documents ordered by BM25 score (highest first).
func (r *SentencesRetriever) KeywordSearch(ctx context.Context, query string, topK int) ([]*schema.Document, error) {
	// Use scoreThreshold of 0 (no filtering) for backward compatibility
	return r.inner.KeywordSearch(ctx, "sentences", query, topK, 0)
}

// GetPool returns the underlying PostgreSQL connection pool.
func (r *SentencesRetriever) GetPool() *pgxpool.Pool {
	return r.inner.GetPool()
}
