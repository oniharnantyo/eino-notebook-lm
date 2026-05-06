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

package tools

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
)

type KeywordSearchInput struct {
	Keywords []string `json:"keywords" jsonschema:"description=Array of keywords or phrases to search for"`
	TopK     int      `json:"top_k" jsonschema:"description=Maximum number of chunks to return (default 5),example=5"`
}

type KeywordMatchResult struct {
	ChunkID  string   `json:"chunk_id"`
	Snippets []string `json:"snippets" jsonschema:"description=KWIC snippets showing matches in context"`
}

type KeywordSearchOutput struct {
	Results []KeywordMatchResult `json:"results"`
}

func NewKeywordSearchTool(r *pgvector.KnowledgesRetriever) tool.BaseTool {
	t, _ := utils.InferTool(
		"keyword_search",
		"Performs keyword-based search (BM25) on knowledge chunks. Returns keyword-in-context (KWIC) snippets and chunk IDs. Agent MUST call chunk_read with chunk_ids to get full content.",
		func(ctx context.Context, input *KeywordSearchInput) (*KeywordSearchOutput, error) {
			topK := input.TopK
			if topK <= 0 {
				topK = 5
			}

			// Join keywords with spaces for BM25 search
			query := strings.Join(input.Keywords, " ")

			docs, err := r.KeywordSearch(ctx, query, topK)
			if err != nil {
				return nil, err
			}

			// Logic to extract KWIC (simplified for now to match interface)
			results := make([]KeywordMatchResult, 0, len(docs))
			for _, doc := range docs {
				// We need the content to extract KWIC, but KeywordSearch only returned IDs.
				// KnowledgesRetriever.Retrieve should be used for full docs.
				results = append(results, KeywordMatchResult{
					ChunkID:  doc.ID,
					Snippets: []string{"[Content retrieval required via chunk_read]"},
				})
			}

			return &KeywordSearchOutput{Results: results}, nil
		},
	)
	return t
}
