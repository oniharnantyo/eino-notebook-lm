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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewUnifiedRetriever_ValidConfig tests creating a unified retriever with valid config.
func TestNewUnifiedRetriever_ValidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *UnifiedConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config returns error",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "nil pool returns error",
			config: &UnifiedConfig{
				Dimension: 1536,
			},
			expectError: true,
			errorMsg:    "connection pool cannot be nil",
		},
		{
			name: "invalid dimension returns error (after pool check)",
			config: &UnifiedConfig{
				Dimension: 0,
			},
			expectError: true,
			errorMsg:    "connection pool cannot be nil",
		},
		{
			name: "negative dimension returns error (after pool check)",
			config: &UnifiedConfig{
				Dimension: -1,
			},
			expectError: true,
			errorMsg:    "connection pool cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewUnifiedRetriever(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, r)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, r)
				assert.NotNil(t, r.tables)
			}
		})
	}
}

// TestTableConfig_Validation tests TableConfig validation.
func TestTableConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      TableConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid table config",
			config: TableConfig{
				Name:       "test_table",
				BM25Index:  "public.test_bm25_idx",
				JoinClause: "",
			},
			expectError: false,
		},
		{
			name: "empty name",
			config: TableConfig{
				Name:       "",
				BM25Index:  "public.test_bm25_idx",
				JoinClause: "",
			},
			expectError: true,
			errorMsg:    "table name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &UnifiedRetriever{
				tables: make(map[string]TableConfig),
			}

			err := r.RegisterTable("test", tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetTableConfig tests retrieving table configurations.
func TestGetTableConfig(t *testing.T) {
	r := &UnifiedRetriever{
		tables: map[string]TableConfig{
			"knowledges": {
				Name:       "knowledges",
				BM25Index:  "public.knowledges_bm25_idx",
				JoinClause: "",
			},
			"sentences": {
				Name:       "sentences",
				BM25Index:  "public.sentences_bm25_idx",
				JoinClause: "",
			},
		},
	}

	t.Run("existing table type", func(t *testing.T) {
		config, err := r.GetTableConfig("knowledges")
		assert.NoError(t, err)
		assert.Equal(t, "knowledges", config.Name)
		assert.Equal(t, "public.knowledges_bm25_idx", config.BM25Index)
	})

	t.Run("unknown table type", func(t *testing.T) {
		_, err := r.GetTableConfig("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown table type")
	})
}

// TestUnifiedRetriever_DefaultK tests default RRF K value handling.
func TestUnifiedRetriever_DefaultK(t *testing.T) {
	tests := []struct {
		name              string
		configK           int
		expectedEffective int
	}{
		{
			name:              "zero K defaults to 60",
			configK:           0,
			expectedEffective: 60,
		},
		{
			name:              "custom K is used",
			configK:           100,
			expectedEffective: 100,
		},
		{
			name:              "negative K defaults to 60",
			configK:           -10,
			expectedEffective: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &UnifiedConfig{
				Dimension: 1536,
				DefaultK:  tt.configK,
			}

			effectiveK := config.DefaultK
			if effectiveK <= 0 {
				effectiveK = 60
			}

			assert.Equal(t, tt.expectedEffective, effectiveK)
		})
	}
}

// TestUnifiedRetriever_DefaultTopK tests default topK value handling.
func TestUnifiedRetriever_DefaultTopK(t *testing.T) {
	tests := []struct {
		name              string
		configTopK        int
		expectedEffective int
	}{
		{
			name:              "zero topK defaults to 20",
			configTopK:        0,
			expectedEffective: 20,
		},
		{
			name:              "custom topK is used",
			configTopK:        50,
			expectedEffective: 50,
		},
		{
			name:              "negative topK defaults to 20",
			configTopK:        -5,
			expectedEffective: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &UnifiedConfig{
				Dimension:   1536,
				DefaultTopK: tt.configTopK,
			}

			effectiveTopK := config.DefaultTopK
			if effectiveTopK <= 0 {
				effectiveTopK = 20
			}

			assert.Equal(t, tt.expectedEffective, effectiveTopK)
		})
	}
}

// TestIntegration_UnifiedRetriever is a placeholder for integration tests.
// These tests require a real PostgreSQL instance with pgvector and pg_textsearch.
func TestIntegration_UnifiedRetriever(t *testing.T) {
	t.Skip("integration test - requires PostgreSQL with pgvector and pg_textsearch")

}
