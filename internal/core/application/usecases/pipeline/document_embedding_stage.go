package pipeline

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
)

// DocumentEmbeddingStage generates embeddings for document chunks.
// This stage is used in the standard pipeline (URL/text content) where
// chunking produces []*schema.Document that need embeddings.
// It stores embeddings in document MetaData["embedding"] for later
// persistence to the Knowledge entity.
type DocumentEmbeddingStage struct {
	embedder embedding.Embedder
}

// NewDocumentEmbeddingStage creates a new DocumentEmbeddingStage.
func NewDocumentEmbeddingStage(embedder embedding.Embedder) *DocumentEmbeddingStage {
	return &DocumentEmbeddingStage{embedder: embedder}
}

func (s *DocumentEmbeddingStage) Name() string { return "DocumentEmbeddingStage" }

func (s *DocumentEmbeddingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for DocumentEmbeddingStage: expected *PipelineData, got %T", input.Data)
	}

	if len(data.Documents) == 0 {
		return StageOutput{Data: data}, nil
	}

	texts := make([]string, len(data.Documents))
	for i, doc := range data.Documents {
		texts[i] = doc.Content
	}

	embeddings, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Store embeddings in document metadata for persistence
	for i, doc := range data.Documents {
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		doc.MetaData["embedding"] = embeddings[i]
	}

	return StageOutput{Data: data}, nil
}
