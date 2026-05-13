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
		metadata := map[string]any{
			"chunk_type":      chunk.ChunkType,
			"first_page":      chunk.Metadata.FirstPage,
			"last_page":       chunk.Metadata.LastPage,
			"byte_start":      chunk.Metadata.ByteStart,
			"byte_end":        chunk.Metadata.ByteEnd,
			"heading_context": chunk.Metadata.HeadingContext,
			"chunk_index":     chunk.Metadata.ChunkIndex,
		}

		k, err := entities.NewKnowledge(
			input.SourceID,
			chunk.Content,
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
