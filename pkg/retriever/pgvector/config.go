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

import "github.com/jackc/pgx/v5/pgxpool"

// DistanceFunction represents the distance function used for vector similarity.
type DistanceFunction string

const (
	// DistanceCosine uses cosine distance for similarity.
	DistanceCosine DistanceFunction = "cosine"

	// DistanceL2 uses Euclidean (L2) distance for similarity.
	DistanceL2 DistanceFunction = "l2"

	// DistanceInnerProduct uses negative inner product for similarity.
	DistanceInnerProduct DistanceFunction = "inner_product"

	// DistanceL1 uses taxicab (L1) distance for similarity.
	DistanceL1 DistanceFunction = "l1"
)

// distanceOperator returns the PostgreSQL operator for the distance function.
func (d DistanceFunction) operator() string {
	switch d {
	case DistanceCosine:
		return "<=>"
	case DistanceL2:
		return "<->"
	case DistanceInnerProduct:
		return "<#>"
	case DistanceL1:
		return "<+>"
	default:
		return "<=>" // Default to cosine
	}
}

// Config holds the configuration for the pgvector retriever.
type Config struct {
	// Pool is the PostgreSQL connection pool.
	Pool *pgxpool.Pool

	// TableName is the name of the table to query documents from.
	// Default: "documents"
	TableName string

	// Dimension is the dimension of the vector embeddings.
	// This must match the embedding model's output dimension.
	Dimension int

	// EmbeddingColumn is the name of the column storing vector embeddings.
	// Default: "embedding"
	EmbeddingColumn string

	// ContentColumn is the name of the column storing document content.
	// Default: "content"
	ContentColumn string

	// MetadataColumn is the name of the column storing document metadata.
	// Default: "metadata"
	MetadataColumn string

	// SubIndexesColumn is the name of the column storing sub-indexes.
	// Default: "sub_indexes"
	SubIndexesColumn string

	// IDColumn is the name of the column storing document IDs.
	// Default: "id"
	IDColumn string

	// DistanceFunction is the distance function to use for vector similarity.
	// Default: DistanceCosine
	DistanceFunction DistanceFunction

	// DefaultTopK is the default number of results to return.
	// Default: 5
	DefaultTopK int

	// DefaultScoreThreshold is the default score threshold for filtering results.
	// Documents with scores below this threshold are excluded.
	// Default: 0.0 (no filtering)
	DefaultScoreThreshold float64
}

// setDefaults sets the default values for the config.
func (c *Config) setDefaults() {
	if c.TableName == "" {
		c.TableName = "documents"
	}
	if c.EmbeddingColumn == "" {
		c.EmbeddingColumn = "embedding"
	}
	if c.ContentColumn == "" {
		c.ContentColumn = "content"
	}
	if c.MetadataColumn == "" {
		c.MetadataColumn = "metadata"
	}
	if c.SubIndexesColumn == "" {
		c.SubIndexesColumn = "sub_indexes"
	}
	if c.IDColumn == "" {
		c.IDColumn = "id"
	}
	if c.DistanceFunction == "" {
		c.DistanceFunction = DistanceCosine
	}
	if c.DefaultTopK <= 0 {
		c.DefaultTopK = 5
	}
}
