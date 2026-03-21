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

// Package pgvector provides a Retriever implementation using PostgreSQL with
// the pgvector extension for vector similarity and hybrid search.
//
// # Overview
//
// The pgvector retriever fetches relevant documents from a PostgreSQL database
// with the pgvector extension. It supports:
//
// - Hybrid search combining BM25 (keyword) and vector (semantic) search
// - Vector similarity search with multiple distance functions
// - Reciprocal Rank Fusion (RRF) for result merging
// - Score threshold filtering
// - Top-K result limiting
// - Sub-index filtering for logical partitioning
// - Metadata and DSL-based filtering
//
// # Hybrid Search
//
// Hybrid search combines BM25 full-text search with vector similarity search
// using Reciprocal Rank Fusion (RRF). This approach provides better retrieval
// quality by leveraging both semantic and keyword-based matching.
//
// ## Prerequisites
//
// Hybrid search requires the pg_textsearch extension for BM25 search.
// You can either set it up manually or enable auto-migration:
//
// ### Manual Setup
//
//	-- Add to postgresql.conf and restart PostgreSQL
//	shared_preload_libraries = 'pg_textsearch'
//
//	-- Enable extension
//	CREATE EXTENSION pg_textsearch;
//
//	-- Create BM25 index on your content column
//	CREATE INDEX idx_knowledges_bm25 ON knowledges
//	USING bm25(content) WITH (text_config='english');
//
// ### Auto-Migration (Recommended for Development)
//
//	Set AutoCreateBM25Extension and AutoCreateBM25Index to true:
//
// ## How Hybrid Search Works
//
// 1. BM25 search retrieves top N candidates using keyword matching
// 2. Vector search retrieves top M candidates using semantic similarity
// 3. Results are merged using RRF: score(d) = Σ (1 / (k + rank(d)))
// 4. Top-K results are returned with RRF scores
//
// The RRF algorithm is robust to score scale differences between BM25 and
// vector search, requiring no weight tuning.
//
// # Usage
//
//	import (
//	    "context"
//	    "github.com/jackc/pgx/v5/pgxpool"
//	    "github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
//	)
//
//	pool, _ := pgxpool.New(context.Background(), "postgres://...")
//	retriever, _ := pgvector.NewRetriever(ctx, &pgvector.Config{
//	    Pool:                    pool,
//	    TableName:               "knowledges",
//	    EmbeddingColumn:         "embedding",
//	    ContentColumn:           "content",
//	    Dimension:               1536,
//	    DistanceFunction:        pgvector.DistanceCosine,
//	    BM25IndexName:           "idx_knowledges_bm25", // For hybrid search
//	    BM25TextConfig:          "english",             // Text search config
//	    AutoCreateBM25Extension: true,                 // Auto-create pg_textsearch extension
//	    AutoCreateBM25Index:     true,                 // Auto-create BM25 index
//	})
//
//	// Retrieve documents using hybrid search
//	docs, _ := retriever.Retrieve(ctx, "what is eino?",
//	    pgvector.WithTopK(5),
//	    pgvector.WithScoreThreshold(0.7),
//	)
//
//	// Customize RRF parameters
//	docs, _ = retriever.Retrieve(ctx, "machine learning basics",
//	    pgvector.WithRRFK(60),              // RRF constant (default: 60)
//	    pgvector.WithBM25Candidates(100),   // BM25 candidates (default: 100)
//	    pgvector.WithVectorCandidates(100), // Vector candidates (default: 100)
//	)
//
// # Configuration Options
//
// ## Config Fields
//
// - BM25IndexName: Name of the pg_textsearch BM25 index (default: "{TableName}_bm25_idx")
// - BM25TextConfig: Text search configuration (default: "english")
// - AutoCreateBM25Extension: Automatically create pg_textsearch extension (default: false)
// - AutoCreateBM25Index: Automatically create BM25 index (default: false)
// - DropBeforeCreate: Drop existing BM25 index before creating (default: false, use with caution)
//
// ## Retrieve Options
//
// - WithRRFK(k): Set RRF constant k (default: 60)
// - WithBM25Candidates(n): Number of BM25 candidates (default: 100)
// - WithVectorCandidates(n): Number of vector candidates (default: 100)
// - WithSkipRerank(skip): Bypass reranker if configured (default: false)
//
// # Distance Functions
//
// The retriever supports multiple distance functions for vector search:
//
// - DistanceCosine (default): Cosine distance
// - DistanceL2: Euclidean (L2) distance
// - DistanceInnerProduct: Negative inner product
// - DistanceL1: Taxicab (L1) distance
//
// Note: RRF scores are stored in document metadata under the "_score" key.
// Higher scores indicate better relevance across both search methods.
package pgvector
