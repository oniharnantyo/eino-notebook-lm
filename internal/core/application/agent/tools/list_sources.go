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
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ListSourcesOutput defines the output structure for the list_sources tool
type ListSourcesOutput struct {
	Sources []SourceDetail `json:"sources"`
}

// SourceDetail provides summary information about a source
type SourceDetail struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ContentType string `json:"content_type"`
}

// ListSourcesInput represents the input for list_sources tool
type ListSourcesInput struct{}

// NewListSourcesTool creates a new ListSourcesTool
func NewListSourcesTool(sourceRepo repositories.SourceRepository, sourceIDs []uuid.UUID) tool.BaseTool {
	t, _ := utils.InferTool(
		"list_sources",
		"Lists the sources available within the current scope, including their ID, title, and content type.",
		func(ctx context.Context, input *ListSourcesInput) (*ListSourcesOutput, error) {
			if len(sourceIDs) == 0 {
				return &ListSourcesOutput{Sources: []SourceDetail{}}, nil
			}

			sources, err := sourceRepo.ListSourceSummariesByID(ctx, sourceIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch sources: %w", err)
			}

			details := make([]SourceDetail, len(sources))
			for i, s := range sources {
				details[i] = SourceDetail{
					ID:          s.ID.String(),
					Title:       s.Title,
					ContentType: string(s.ContentType),
				}
			}

			return &ListSourcesOutput{Sources: details}, nil
		},
	)
	return t
}
