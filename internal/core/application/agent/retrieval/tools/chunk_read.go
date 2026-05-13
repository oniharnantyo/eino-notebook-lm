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
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type ChunkReadInput struct {
	ChunkIDs []string `json:"chunk_ids" jsonschema:"description=Array of parent chunk IDs (chunk_id field) from search results."`
}

type ChunkResult struct {
	ChunkID string `json:"chunk_id"`
	Content string `json:"content,omitempty"`
	Status  string `json:"status,omitempty"`
}

type ChunkReadOutput struct {
	Chunks []ChunkResult `json:"chunks"`
}

// IContextTracker is a minimal interface for the context tracker to avoid direct package dependency.
type IContextTracker interface {
	ReadOrMark(id string) bool
}

func NewChunkReadTool(repo repositories.KnowledgeRepository, tracker IContextTracker) tool.BaseTool {
	t, _ := utils.InferTool(
		"chunk_read",
		"Reads the full content of multiple knowledge chunks by their IDs. Useful when search result snippets indicate relevant information but are too short. Agent MUST call this to get full content before finalizing an answer if snippets are insufficient.",
		func(ctx context.Context, input *ChunkReadInput) (*ChunkReadOutput, error) {
			results := make([]ChunkResult, 0, len(input.ChunkIDs))
			var unreadIDs []uuid.UUID
			idMap := make(map[string]int)

			for _, idStr := range input.ChunkIDs {
				if tracker != nil && tracker.ReadOrMark(idStr) {
					results = append(results, ChunkResult{
						ChunkID: idStr,
						Status:  "Already read",
					})
					continue
				}

				id, err := uuid.Parse(idStr)
				if err != nil {
					results = append(results, ChunkResult{
						ChunkID: idStr,
						Status:  "Invalid ID format",
					})
					continue
				}

				idMap[idStr] = len(results)
				unreadIDs = append(unreadIDs, id)
				results = append(results, ChunkResult{ChunkID: idStr}) // placeholder
			}

			if len(unreadIDs) > 0 {
				knowledges, err := repo.FindByIDs(ctx, unreadIDs)
				if err != nil {
					return nil, err
				}

				foundMap := make(map[string]*entities.Knowledge)
				for _, k := range knowledges {
					foundMap[k.ID.String()] = k
				}

				for idStr, idx := range idMap {
					if k, ok := foundMap[idStr]; ok {
						results[idx].Content = k.Content
						results[idx].Status = "Success"
					} else {
						results[idx].Status = "Not found"
					}
				}
			}

			return &ChunkReadOutput{Chunks: results}, nil
		},
	)
	return t
}
