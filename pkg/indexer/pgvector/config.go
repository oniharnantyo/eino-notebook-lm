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
	// Cosine distance = 1 - cosine similarity.
	DistanceCosine DistanceFunction = "cosine"

	// DistanceL2 uses Euclidean (L2) distance for similarity.
	DistanceL2 DistanceFunction = "l2"

	// DistanceInnerProduct uses negative inner product for similarity.
	// PostgreSQL returns negative inner product since it only supports ASC order.
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

// indexOperator returns the PostgreSQL index operator class for the distance function.
func (d DistanceFunction) indexOperator() string {
	switch d {
	case DistanceCosine:
		return "vector_cosine_ops"
	case DistanceL2:
		return "vector_l2_ops"
	case DistanceInnerProduct:
		return "vector_ip_ops"
	case DistanceL1:
		return "vector_l1_ops"
	default:
		return "vector_cosine_ops"
	}
}

// Config holds the configuration for the pgvector indexer.
type Config struct {
	// Pool is the PostgreSQL connection pool.
	Pool *pgxpool.Pool

	// TableName is the name of the table to store knowledges.
	// Default: "knowledges"
	TableName string

	// TableSchema is the schema name for the table.
	// Default: "public"
	TableSchema string

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
	// Default: "knowledge_id"
	IDColumn string

	// ReferenceIDColumn is the name of the column storing a reference ID (e.g., notebook_id, user_id).
	// The value is extracted from the "reference_id" key in document metadata.
	// Default: "" (disabled)
	// Set to "notebook_id" to enable reference ID tracking.
	ReferenceIDColumn string

	// ReferenceIDIndexName is the name of the index to create on the ReferenceIDColumn.
	// Default: "{tableName}_{referenceIDColumn}_idx"
	ReferenceIDIndexName string

	// CreatedAtColumn is the name of the column storing creation timestamp.
	// Default: "created_at"
	CreatedAtColumn string

	// UpdatedAtColumn is the name of the column storing update timestamp.
	// Default: "updated_at"
	UpdatedAtColumn string

	// DocumentIDColumn is the name of the column storing the document ID from schema.Document.
	// Default: "document_id"
	DocumentIDColumn string

	// DistanceFunction is the distance function to use for vector similarity.
	// Default: DistanceCosine
	DistanceFunction DistanceFunction

	// CreateIndexIfNotExists creates a HNSW index if it doesn't exist.
	// Default: false
	CreateIndexIfNotExists bool

	// IndexName is the name of the index to create.
	// Default: "{tableName}_embedding_idx"
	IndexName string

	// AutoCreateTable creates the documents table if it doesn't exist.
	// Default: false
	AutoCreateTable bool

	// AutoCreateExtension creates the vector extension if it doesn't exist.
	// Default: false
	AutoCreateExtension bool

	// HNSWM is the number of bi-directional links for each node in the HNSW index.
	// Higher values improve recall but increase memory usage and build time.
	// Typical range: 4-64. Default: 16
	HNSWM int

	// HNSWEFConstruction is the size of the dynamic candidate list for construction.
	// Higher values improve index quality but increase build time.
	// Typical range: 8-512. Default: 64
	HNSWEFConstruction int

	// SearchEF is the size of the dynamic candidate list for search.
	// Higher values improve recall but decrease search speed.
	// If 0, uses HNSWM. Default: 0 (use HNSWM)
	SearchEF int

	// DefaultK is the default number of results to return for similarity searches.
	// Default: 10
	DefaultK int

	// IncludeDistance in search results includes the distance/score in the document metadata.
	// Default: true
	IncludeDistance bool

	// UseIVFFlat creates an IVFFlat index instead of HNSW.
	// IVFFlat is better for exact search but slower for high-dimensional data.
	// Default: false (use HNSW)
	UseIVFFlat bool

	// IVFLists is the number of lists for IVFFlat index.
	// Should be sqrt(rows) for optimal performance.
	// Default: 100
	IVFLists int

	// DropBeforeCreate drops the table/index before creating if AutoCreateTable/CreateIndexIfNotExists is set.
	// Use with caution as this will delete existing data.
	// Default: false
	DropBeforeCreate bool

	// BatchSize is the number of documents to process in a single batch.
	// Default: 100
	BatchSize int

	// UseJSONBForMetadata stores metadata as JSONB (false uses JSON).
	// JSONB is more efficient for querying but slightly slower to insert.
	// Default: true
	UseJSONBForMetadata bool

	// EnableSubIndexes enables filtering by sub-indexes.
	// Default: true
	EnableSubIndexes bool
}

// setDefaults sets the default values for the config.
func (c *Config) setDefaults() {
	if c.TableName == "" {
		c.TableName = "knowledges"
	}
	if c.TableSchema == "" {
		c.TableSchema = "public"
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
	if c.ReferenceIDColumn != "" && c.ReferenceIDIndexName == "" {
		c.ReferenceIDIndexName = c.TableName + "_" + c.ReferenceIDColumn + "_idx"
	}

	if c.CreatedAtColumn == "" {
		c.CreatedAtColumn = "created_at"
	}
	if c.UpdatedAtColumn == "" {
		c.UpdatedAtColumn = "updated_at"
	}
	if c.DocumentIDColumn == "" {
		c.DocumentIDColumn = "document_id"
	}
	if c.DistanceFunction == "" {
		c.DistanceFunction = DistanceCosine
	}
	if c.IndexName == "" {
		c.IndexName = c.TableName + "_embedding_idx"
	}
	if c.HNSWM == 0 {
		c.HNSWM = 16
	}
	if c.HNSWEFConstruction == 0 {
		c.HNSWEFConstruction = 64
	}
	if c.DefaultK == 0 {
		c.DefaultK = 10
	}
	if c.IVFLists == 0 {
		c.IVFLists = 100
	}
	if c.BatchSize == 0 {
		c.BatchSize = 100
	}
}
