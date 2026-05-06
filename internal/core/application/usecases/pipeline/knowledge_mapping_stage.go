package pipeline

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// KnowledgeMappingStage converts extraction results into domain knowledge entities.
type KnowledgeMappingStage struct {
	logger *logger.Logger
}

// NewKnowledgeMappingStage creates a new KnowledgeMappingStage.
func NewKnowledgeMappingStage(log *logger.Logger) *KnowledgeMappingStage {
	return &KnowledgeMappingStage{
		logger: log,
	}
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
		s.logger.Error("ExtractionResult is nil in PipelineData", "source_id", input.SourceID)
		return StageOutput{}, fmt.Errorf("ExtractionResult is nil in PipelineData")
	}

	s.logger.Info("Mapping chunks to knowledges", "source_id", input.SourceID, "count", len(data.ExtractionResult.Chunks))
	knowledges := make([]*entities.Knowledge, 0, len(data.ExtractionResult.Chunks))
	for _, chunk := range data.ExtractionResult.Chunks {
		metadata := make(map[string]any)

		// Set chunk-level metadata first
		metadata["chunk_type"] = chunk.ChunkType

		// Merge document-level metadata (takes precedence)
		if data.ExtractionResult.Metadata != nil {
			for k, v := range data.ExtractionResult.Metadata {
				metadata[k] = v
			}
		}

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
			s.logger.Error("Failed to create knowledge entity", "source_id", input.SourceID, "error", err)
			return StageOutput{}, fmt.Errorf("failed to create knowledge entity: %w", err)
		}

		knowledges = append(knowledges, k)
	}

	data.Knowledges = knowledges
	s.logger.Info("Successfully mapped chunks to knowledges", "source_id", input.SourceID, "count", len(knowledges))

	return StageOutput{Data: data}, nil
}
