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

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
)

type SemanticSearchInput struct {
	Query string `json:"query" jsonschema:"description=The semantic search query to find relevant snippets based on meaning"`
	TopK  int    `json:"top_k" jsonschema:"description=Maximum number of snippets to return (default 5),example=5"`
}

type SemanticSearchResult struct {
	ChunkID string  `json:"chunk_id"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

type SemanticSearchOutput struct {
	Results []SemanticSearchResult `json:"results"`
}

func NewSemanticSearchTool(r *pgvector.SentencesRetriever, embedder embedding.Embedder) tool.BaseTool {
	t, _ := utils.InferTool(
		"semantic_search",
		"Performs semantic similarity search to find relevant snippets.",
		func(ctx context.Context, input *SemanticSearchInput) (*SemanticSearchOutput, error) {
			topK := input.TopK
			if topK <= 0 {
				topK = 5
			}

			// Generate vector for the query
			vecs, err := embedder.EmbedStrings(ctx, []string{input.Query})
			if err != nil {
				return nil, err
			}

			docs, err := r.SemanticSearch(ctx, vecs[0], topK)
			if err != nil {
				return nil, err
			}

			results := make([]SemanticSearchResult, 0, len(docs))
			for _, doc := range docs {
				results = append(results, SemanticSearchResult{
					ChunkID: doc.ID,
					Snippet: "[Content retrieval required via chunk_read]",
					Score:   0.0, // Temporary: replace Rank with proper score access if available
				})
			}

			return &SemanticSearchOutput{Results: results}, nil
		},
	)
	return t
}
