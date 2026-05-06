package pipeline

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/embedding"
)

type EmbeddingStage struct {
	embedder embedding.Embedder
}

func NewEmbeddingStage(embedder embedding.Embedder) *EmbeddingStage {
	return &EmbeddingStage{embedder: embedder}
}

func (s *EmbeddingStage) Name() string { return "EmbeddingStage" }

func (s *EmbeddingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for EmbeddingStage: expected *PipelineData, got %T", input.Data)
	}

	if len(data.Sentences) == 0 {
		return StageOutput{Data: data}, nil
	}

	texts := make([]string, len(data.Sentences))
	for i, sentence := range data.Sentences {
		texts[i] = sentence.Content
	}

	embeddings, err := s.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Attach embedding vectors to each sentence in place
	for i := range data.Sentences {
		data.Sentences[i].Embedding = ConvertFloat64ToFloat32(embeddings[i])
	}

	return StageOutput{Data: data}, nil
}
