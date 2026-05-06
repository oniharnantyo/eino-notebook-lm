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
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/tool"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/retriever/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ScopeConfig defines the scope for retrieval tools in a single agent run.
type ScopeConfig struct {
	SourceIDs   []uuid.UUID
	SourceTypes []string
	Tracker     IContextTracker
}

// ToolFactory creates scoped retrieval tools for the agent.
type ToolFactory struct {
	retriever          *pgvector.SentencesRetriever
	imageRetriever     *pgvector.ImagesRetriever
	knowledgeRetriever *pgvector.KnowledgesRetriever
	knowledgeRepo      repositories.KnowledgeRepository
	sourceRepo         repositories.SourceRepository
	embedder           embedding.Embedder
}

// NewToolFactory creates a new ToolFactory.
func NewToolFactory(r *pgvector.SentencesRetriever, ir *pgvector.ImagesRetriever, kr *pgvector.KnowledgesRetriever, k repositories.KnowledgeRepository, s repositories.SourceRepository, e embedding.Embedder) *ToolFactory {
	return &ToolFactory{
		retriever:          r,
		imageRetriever:     ir,
		knowledgeRetriever: kr,
		knowledgeRepo:      k,
		sourceRepo:         s,
		embedder:           e,
	}
}

// NewScopedTools returns a set of tools scoped to the provided config.
func (f *ToolFactory) NewScopedTools(cfg ScopeConfig) []tool.BaseTool {
	return []tool.BaseTool{
		NewKeywordSearchTool(f.knowledgeRetriever),
		NewSemanticSearchTool(f.retriever, f.embedder),
		NewImageSearchTool(f.imageRetriever),
		NewChunkReadTool(f.knowledgeRepo, cfg.Tracker),
		NewListSourcesTool(f.sourceRepo, cfg.SourceIDs),
	}
}

// IsSourceTypeSupported checks if the given source type is supported by the factory's retrievers.
func (f *ToolFactory) IsSourceTypeSupported(sourceType string) bool {
	switch sourceType {
	case "image":
		return f.imageRetriever != nil
	case "knowledge":
		return f.knowledgeRetriever != nil
	case "sentence", "pdf", "text", "docx", "website":
		return f.retriever != nil
	default:
		// By default, if we have a general retriever, we assume it can handle filtering by unknown types
		return f.retriever != nil
	}
}
