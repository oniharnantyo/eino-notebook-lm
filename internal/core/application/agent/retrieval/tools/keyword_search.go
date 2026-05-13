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

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
)

type KeywordSearchInput struct {
	Keywords []string `json:"keywords" jsonschema:"description=MUST be an array of keyword strings, e.g. [\"keyword1\", \"keyword2\"]. Do not pass a single string."`
	TopK     int      `json:"top_k" jsonschema:"description=Maximum number of chunks to return (default 5),example=5"`
}

func NewKeywordSearchTool(r *pgvector.KnowledgesRetriever) tool.BaseTool {
	t, _ := utils.InferTool(
		"keyword_search",
		"Performs keyword-based search (BM25) on knowledge chunks. Returns keyword-in-context (KWIC) snippets and chunk IDs. Agent MUST call chunk_read with chunk_ids to get full content.",
		func(ctx context.Context, input *KeywordSearchInput) ([]*schema.Document, error) {
			topK := input.TopK
			if topK <= 0 {
				topK = 5
			}

			return r.KeywordSearch(ctx, input.Keywords, topK)
		},
	)
	return t
}
