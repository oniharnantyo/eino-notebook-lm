package pipeline

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// KnowledgeMappingStage converts extraction results into domain knowledge entities.
type KnowledgeMappingStage struct{}

// NewKnowledgeMappingStage creates a new KnowledgeMappingStage.
func NewKnowledgeMappingStage() *KnowledgeMappingStage {
	return &KnowledgeMappingStage{}
}

// Name returns "KnowledgeMappingStage".
func (s *KnowledgeMappingStage) Name() string {
	return "KnowledgeMappingStage"
}

// Execute maps extraction chunks to knowledge entities.
// Input: *PipelineData
// Output: *PipelineData with Knowledges populated
func (s *KnowledgeMappingStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for KnowledgeMappingStage: expected *PipelineData, got %T", input.Data)
	}

	if data.ExtractionResult == nil {
		return StageOutput{}, fmt.Errorf("ExtractionResult is nil in PipelineData")
	}

	knowledges := make([]*entities.Knowledge, 0, len(data.ExtractionResult.Chunks))
	for _, chunk := range data.ExtractionResult.Chunks {
		metadata := make(map[string]any)
		metadata["chunk_type"] = chunk.ChunkType

		k, err := entities.NewKnowledge(
			input.SourceID,
			chunk.Content,
			chunk.Metadata.ChunkIndex,
			chunk.Metadata.HeadingContext,
			chunk.Metadata.FirstPage,
			chunk.Metadata.LastPage,
			metadata,
		)
		if err != nil {
			return StageOutput{}, fmt.Errorf("failed to create knowledge entity: %w", err)
		}

		knowledges = append(knowledges, k)
	}

	data.Knowledges = knowledges

	return StageOutput{Data: data}, nil
}
