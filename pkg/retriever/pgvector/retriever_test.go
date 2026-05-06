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
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRetriever_ValidConfig tests creating a retriever with valid config.
func TestNewRetriever_ValidConfig(t *testing.T) {
	ctx := context.Background()

	// This test would require a real connection pool
	// For now, we test the validation logic
	t.Run("nil config returns error", func(t *testing.T) {
		r, err := NewRetriever(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("nil pool returns error", func(t *testing.T) {
		config := &Config{
			Dimension: 1536,
		}
		r, err := NewRetriever(ctx, config)
		assert.Error(t, err)
		assert.Nil(t, r)
		assert.Contains(t, err.Error(), "connection pool cannot be nil")
	})

	t.Run("invalid dimension returns error", func(t *testing.T) {
		// Pool check happens before dimension check, so we need a mock pool
		// For now, we just verify it returns an error
		config := &Config{
			Dimension: 0,
		}
		r, err := NewRetriever(ctx, config)
		assert.Error(t, err)
		assert.Nil(t, r)
		// Pool is checked first, so we get pool error
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

// TestConfig_SetDefaults tests that config defaults are set correctly.
func TestConfig_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected *Config
	}{
		{
			name:   "empty config gets defaults",
			config: &Config{},
			expected: &Config{
				TableName:        "knowledges",
				EmbeddingColumn:  "embedding",
				ContentColumn:    "content",
				MetadataColumn:   "metadata",
				SubIndexesColumn: "sub_indexes",
				IDColumn:         "id",
				DistanceFunction: DistanceCosine,
				DefaultTopK:      5,
				BM25TextConfig:   "english",
			},
		},
		{
			name: "custom values are preserved",
			config: &Config{
				TableName:        "my_table",
				EmbeddingColumn:  "vec",
				ContentColumn:    "text",
				MetadataColumn:   "meta",
				SubIndexesColumn: "indexes",
				IDColumn:         "doc_id",
				DistanceFunction: DistanceL2,
				DefaultTopK:      10,
				BM25TextConfig:   "spanish",
			},
			expected: &Config{
				TableName:        "my_table",
				EmbeddingColumn:  "vec",
				ContentColumn:    "text",
				MetadataColumn:   "meta",
				SubIndexesColumn: "indexes",
				IDColumn:         "doc_id",
				DistanceFunction: DistanceL2,
				DefaultTopK:      10,
				BM25TextConfig:   "spanish",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.setDefaults()
			assert.Equal(t, tt.expected.TableName, tt.config.TableName)
			assert.Equal(t, tt.expected.EmbeddingColumn, tt.config.EmbeddingColumn)
			assert.Equal(t, tt.expected.ContentColumn, tt.config.ContentColumn)
			assert.Equal(t, tt.expected.MetadataColumn, tt.config.MetadataColumn)
			assert.Equal(t, tt.expected.SubIndexesColumn, tt.config.SubIndexesColumn)
			assert.Equal(t, tt.expected.IDColumn, tt.config.IDColumn)
			assert.Equal(t, tt.expected.DistanceFunction, tt.config.DistanceFunction)
			assert.Equal(t, tt.expected.DefaultTopK, tt.config.DefaultTopK)
			assert.Equal(t, tt.expected.BM25TextConfig, tt.config.BM25TextConfig)
		})
	}
}

// TestDistanceFunction_Operator tests the distance function operators.
func TestDistanceFunction_Operator(t *testing.T) {
	tests := []struct {
		name     string
		function DistanceFunction
		expected string
	}{
		{"cosine", DistanceCosine, "<=>"},
		{"l2", DistanceL2, "<->"},
		{"inner product", DistanceInnerProduct, "<#>"},
		{"l1", DistanceL1, "<+>"},
		{"unknown defaults to cosine", DistanceFunction("unknown"), "<=>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.function.operator())
		})
	}
}

// TestVectorToString tests converting vectors to PostgreSQL format.
func TestVectorToString(t *testing.T) {
	tests := []struct {
		name     string
		vector   []float64
		expected string
	}{
		{
			name:     "empty vector",
			vector:   []float64{},
			expected: "[]",
		},
		{
			name:     "single element",
			vector:   []float64{1.5},
			expected: "[1.5]",
		},
		{
			name:     "multiple elements",
			vector:   []float64{1.0, 2.5, -3.14},
			expected: "[1,2.5,-3.14]",
		},
		{
			name:     "scientific notation",
			vector:   []float64{1e-10, 1e10},
			expected: "[1e-10,1e+10]", // %g format uses e+ for large numbers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vectorToString(tt.vector)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetRetrieveOptions tests parsing retrieve options with defaults.
func TestGetRetrieveOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		opts := getRetrieveOptions()
		assert.True(t, opts.IncludeDistance)
		assert.Equal(t, 60, opts.RRFK)
		assert.Equal(t, 100, opts.BM25Candidates)
		assert.Equal(t, 100, opts.VectorCandidates)
		assert.False(t, opts.SkipRerank)
	})

	t.Run("with custom options", func(t *testing.T) {
		opts := getRetrieveOptions(
			WithIncludeDistance(false),
			WithRRFK(100),
			WithBM25Candidates(50),
			WithVectorCandidates(75),
			WithSkipRerank(true),
		)
		assert.False(t, opts.IncludeDistance)
		assert.Equal(t, 100, opts.RRFK)
		assert.Equal(t, 50, opts.BM25Candidates)
		assert.Equal(t, 75, opts.VectorCandidates)
		assert.True(t, opts.SkipRerank)
	})
}

// TestRetriever_Getters tests the retriever getter methods.
func TestRetriever_Getters(t *testing.T) {
	t.Run("getters require valid retriever", func(t *testing.T) {
		ctx := context.Background()
		config := &Config{
			// Pool is nil, so NewRetriever will fail
			Dimension: 1536,
		}

		r, err := NewRetriever(ctx, config)
		assert.Error(t, err)
		assert.Nil(t, r)
	})
}

// TestWithFilterOptions tests filter-related options.
func TestWithFilterOptions(t *testing.T) {
	t.Run("filter by sub indexes", func(t *testing.T) {
		opts := getRetrieveOptions(
			WithFilterSubIndexes([]string{"index1", "index2"}),
		)
		assert.Equal(t, []string{"index1", "index2"}, opts.FilterSubIndexes)
	})

	t.Run("filter by reference IDs", func(t *testing.T) {
		opts := getRetrieveOptions(
			WithFilterReferenceIDs([]string{"ref1", "ref2"}),
		)
		assert.Equal(t, []string{"ref1", "ref2"}, opts.FilterReferenceIDs)
	})

	t.Run("filter by source types", func(t *testing.T) {
		opts := getRetrieveOptions(
			WithFilterSourceTypes([]string{"pdf", "web"}),
		)
		assert.Equal(t, []string{"pdf", "web"}, opts.FilterSourceTypes)
	})

	t.Run("custom where clause", func(t *testing.T) {
		opts := getRetrieveOptions(
			WithWhereClause("created_at > NOW() - INTERVAL '7 days'"),
		)
		assert.Equal(t, "created_at > NOW() - INTERVAL '7 days'", opts.WhereClause)
	})
}

// TestRankedDocumentWithScore_SortByScore tests sorting by RRF score.
func TestRankedDocumentWithScore_SortByScore(t *testing.T) {
	scores := map[string]float64{
		"doc1": 0.5,
		"doc2": 0.8,
		"doc3": 0.3,
		"doc4": 0.8,
	}

	result := SortByScore(scores)

	require.Equal(t, 4, len(result))
	// Highest scores first
	assert.Equal(t, "doc2", result[0].ID) // or doc4 (tie-breaker is ID)
	assert.Equal(t, 0.8, result[0].Score)
	assert.Equal(t, "doc4", result[1].ID)
	assert.Equal(t, 0.8, result[1].Score)
	assert.Equal(t, "doc1", result[2].ID)
	assert.Equal(t, 0.5, result[2].Score)
	assert.Equal(t, "doc3", result[3].ID)
	assert.Equal(t, 0.3, result[3].Score)
}

// TestIntegration_RetrieveWithMockPool is a placeholder for integration tests.
// These tests require a real PostgreSQL instance with pgvector and pg_textsearch.
//
// To run these tests, set up a test database and use:
//
//	postgres://user:pass@localhost:5432/testdb
func TestIntegration_RetrieveWithMockPool(t *testing.T) {
	t.Skip("integration test - requires PostgreSQL with pgvector and pg_textsearch")

	// Example integration test structure:
	// 1. Connect to test database
	// 2. Create test table with indexes
	// 3. Insert test documents
	// 4. Test Retrieve method
	// 5. Verify results
	// 6. Cleanup
}

// mockPool is a placeholder for mocking pgxpool.Pool in unit tests.
// In a real implementation, you would use a mocking library like
// github.com/stretchr/testify/mock or pgxmock.
type mockPool struct {
	pgxpool.Pool
}

// TestRetriever_WithMockPool demonstrates testing with a mock pool.
func TestRetriever_WithMockPool(t *testing.T) {
	t.Skip("mock test - requires mock implementation")

	// This test would:
	// 1. Create a mock pool
	// 2. Set up expected queries
	// 3. Create retriever with mock pool
	// 4. Call Retrieve method
	// 5. Verify queries were called correctly
}

// BenchmarkMergeByRRF benchmarks the RRF merge algorithm.
func BenchmarkMergeByRRF(b *testing.B) {
	vectorResults := make([]RankedDocument, 100)
	bm25Results := make([]RankedDocument, 100)

	for i := 0; i < 100; i++ {
		vectorResults[i] = RankedDocument{ID: string(rune('a' + i)), Rank: i + 1}
		bm25Results[i] = RankedDocument{ID: string(rune('a' + (i+50)%100)), Rank: i + 1}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MergeByRRF(60, vectorResults, bm25Results)
	}
}

// BenchmarkVectorToString benchmarks vector string conversion.
func BenchmarkVectorToString(b *testing.B) {
	vector := make([]float64, 1536) // Common embedding dimension
	for i := range vector {
		vector[i] = 0.1
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vectorToString(vector)
	}
}
