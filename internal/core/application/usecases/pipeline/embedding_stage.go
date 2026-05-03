package pipeline

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
)

type EmbeddingStage struct {
	embedder embedding.Embedder
}

func NewEmbeddingStage(embedder embedding.Embedder) *EmbeddingStage {
	return &EmbeddingStage{embedder: embedder}
}

func (s *EmbeddingStage) Name() string { return "EmbeddingStage" }

func (s *EmbeddingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	docs, ok := input.Data.([]*schema.Document)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for EmbeddingStage: expected []*schema.Document, got %T", input.Data)
	}

	if len(docs) == 0 {
		return StageOutput{Data: docs}, nil
	}

	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	embeddings, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	for i, doc := range docs {
		if doc.MetaData == nil {
			doc.MetaData = make(map[string]any)
		}
		doc.MetaData["embedding"] = embeddings[i]
	}

	return StageOutput{Data: docs}, nil
}
