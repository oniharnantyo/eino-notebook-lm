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
	"encoding/json"
	"fmt"
	"strings"
)

// BM25Result represents a document with its BM25 score.
type BM25Result struct {
	ID       string
	Score    float64
	Content  string
	Metadata map[string]any
}

// executeBM25Search performs BM25 full-text search using the pg_textsearch extension.
//
// The pg_textsearch extension provides native BM25 scoring through the <@> operator.
// Scores are negative (lower is better), so we filter for scores < 0 to ensure
// we only get relevant results.
func (r *Retriever) executeBM25Search(
	ctx context.Context,
	query string,
	retrieveOpts *RetrieveOptions,
	limit int,
) ([]BM25Result, error) {
	var whereClauses []string
	var args []any
	argPos := 1

	// Add reference ID filtering using the dedicated column
	if r.config.ReferenceIDColumn != "" && len(retrieveOpts.FilterReferenceIDs) > 0 {
		placeholders := make([]string, len(retrieveOpts.FilterReferenceIDs))
		for i, id := range retrieveOpts.FilterReferenceIDs {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, id)
			argPos++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("%s IN (%s)",
			r.config.ReferenceIDColumn, strings.Join(placeholders, ", ")))
	}

	// Add source_type filtering using parameterized query
	if len(retrieveOpts.FilterSourceTypes) > 0 {
		placeholders := make([]string, len(retrieveOpts.FilterSourceTypes))
		for i, sourceType := range retrieveOpts.FilterSourceTypes {
			placeholders[i] = fmt.Sprintf("$%d", argPos)
			args = append(args, sourceType)
			argPos++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("metadata->>'source_type' IN (%s)",
			strings.Join(placeholders, ", ")))
	}

	// Add custom WHERE clause
	if retrieveOpts.WhereClause != "" {
		whereClauses = append(whereClauses, retrieveOpts.WhereClause)
	}

	// Build WHERE clause
	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "AND " + strings.Join(whereClauses, " AND ")
	}

	// Get BM25 index name - must be a string literal in the query, not a parameter
	// because to_bm25query needs to look up index metadata at plan time.
	// This is a PostgreSQL requirement for functions that inspect schema objects.
	indexName := r.config.BM25IndexName
	if indexName == "" {
		// Use a default index name convention (schema-qualified for public schema)
		indexName = "public." + r.config.TableName + "_bm25_idx"
	} else if !strings.Contains(indexName, ".") {
		// If index name is not schema-qualified, add public. prefix
		// This is required for to_bm25query to find the index
		indexName = "public." + indexName
	}

	// Build the BM25 query using pg_textsearch
	// The <@> operator returns negative scores (lower = better)
	// We filter with < 0 to ensure relevant results
	// IMPORTANT: Both index name and query string are interpolated directly into SQL
	// This is required because to_bm25query() needs these at plan time, not execution time.
	// The query is safely escaped for single quotes to prevent SQL injection.
	// Note: to_bm25query() parses the query for BM25 search terms, not SQL execution.
	escapedQuery := strings.ReplaceAll(query, "'", "''") // Escape single quotes

	bm25Query := fmt.Sprintf(`
		SELECT
			%s,
			%s,
			%s,
			%s <@> '%s' AS bm25_score
		FROM %s
		WHERE %s <@> to_bm25query('%s', '%s') < 0
			%s
		ORDER BY %s <@> '%s'
		LIMIT $%d
	`,
		r.config.IDColumn,
		r.config.ContentColumn,
		r.config.MetadataColumn,
		r.config.ContentColumn,
		escapedQuery, // query as string literal in SELECT
		r.config.TableName,
		r.config.ContentColumn,
		escapedQuery, // query as string literal in WHERE to_bm25query
		indexName,    // index name as string literal
		whereSQL,
		r.config.ContentColumn,
		escapedQuery, // query as string literal in ORDER BY
		argPos,       // LIMIT uses the next available parameter index
	)

	// Add limit
	args = append(args, limit)

	// Execute query
	rows, err := r.pool.Query(ctx, bm25Query, args...)
	if err != nil {
		return nil, fmt.Errorf("bm25 query failed: %w", err)
	}
	defer rows.Close()

	// Parse results
	var results []BM25Result
	for rows.Next() {
		var id, content string
		var metadataJSONB []byte
		var bm25Score float64

		if err := rows.Scan(&id, &content, &metadataJSONB, &bm25Score); err != nil {
			return nil, fmt.Errorf("failed to scan bm25 row: %w", err)
		}

		// Parse metadata
		var metadata map[string]any
		if metadataJSONB != nil {
			if err := json.Unmarshal(metadataJSONB, &metadata); err != nil {
				// If unmarshaling fails, continue with empty metadata
				metadata = make(map[string]any)
			}
		}

		results = append(results, BM25Result{
			ID:       id,
			Score:    bm25Score,
			Content:  content,
			Metadata: metadata,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bm25 row iteration error: %w", err)
	}

	return results, nil
}
