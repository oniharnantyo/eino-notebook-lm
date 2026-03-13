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

// Package pgvector provides an Indexer implementation using PostgreSQL with
// the pgvector extension for vector similarity search.
//
// # Overview
//
// The pgvector indexer stores documents and their vector embeddings in a
// PostgreSQL database with the pgvector extension. It supports:
//
// - Exact and approximate nearest neighbor search
// - Multiple distance functions (L2, inner product, cosine, L1)
// - Sub-indexes for logical partitioning
// - Automatic embedding generation via Embedder
//
// # Database Setup
//
// Enable the pgvector extension in your database:
//
//	CREATE EXTENSION IF NOT EXISTS vector;
//
// Create a table for storing documents:
//
//	CREATE TABLE documents (
//	    id TEXT PRIMARY KEY,
//	    content TEXT NOT NULL,
//	    embedding vector(1536),
//	    metadata JSONB,
//	    sub_indexes TEXT[],
//	    created_at TIMESTAMP DEFAULT NOW()
//	);
//
// Create an index for efficient vector search:
//
//	CREATE INDEX documents_embedding_idx ON documents
//	    USING hnsw (embedding vector_cosine_ops);
//
// # Usage
//
//	import (
//	    "context"
//	    "github.com/jackc/pgx/v5/pgxpool"
//	    "github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
//	)
//
//	pool, _ := pgxpool.New(context.Background(), "postgres://...")
//	indexer, _ := pgvector.NewIndexer(ctx, &pgvector.Config{
//	    Pool:       pool,
//	    TableName:  "documents",
//	    Dimension:  1536,
//	})
//
//	// Store documents with automatic embedding
//	docs := []*schema.Document{{Content: "Hello world"}}
//	ids, _ := indexer.Store(ctx, docs, indexer.WithEmbedding(embedder))
//
// # Distance Functions
//
// The indexer supports multiple distance functions for vector similarity:
//
// - DistanceCosine (default): Cosine distance
// - DistanceL2: Euclidean (L2) distance
// - DistanceInnerProduct: Negative inner product
// - DistanceL1: Taxicab (L1) distance
//
// Use WithDistanceFunction option to configure:
//
//	indexer, _ := pgvector.NewIndexer(ctx, &pgvector.Config{
//	    Pool:              pool,
//	    TableName:         "documents",
//	    Dimension:         1536,
//	    DistanceFunction:  pgvector.DistanceL2,
//	})
package pgvector
