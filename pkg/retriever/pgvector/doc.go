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
// the pgvector extension for vector similarity search.
//
// # Overview
//
// The pgvector retriever fetches relevant documents from a PostgreSQL database
// with the pgvector extension. It supports:
//
// - Vector similarity search with multiple distance functions
// - Score threshold filtering
// - Top-K result limiting
// - Sub-index filtering for logical partitioning
// - Metadata and DSL-based filtering
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
//	    Pool:             pool,
//	    TableName:        "knowledges",
//	    EmbeddingColumn:  "embedding",
//	    ContentColumn:    "content",
//	    Dimension:        1536,
//	    DistanceFunction: pgvector.DistanceCosine,
//	})
//
//	// Retrieve documents
//	docs, _ := retriever.Retrieve(ctx, "what is eino?",
//	    pgvector.WithTopK(5),
//	    pgvector.WithScoreThreshold(0.7),
//	)
//
// # Distance Functions
//
// The retriever supports multiple distance functions:
//
// - DistanceCosine (default): Cosine distance
// - DistanceL2: Euclidean (L2) distance
// - DistanceInnerProduct: Negative inner product
// - DistanceL1: Taxicab (L1) distance
//
// Note: Distance scores are stored in document metadata under the "_score" key.
// For cosine similarity, use (1 - distance) to get the actual similarity score.
package pgvector
