package pipeline

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
)

// ExtractionStage extracts raw content from various sources (files, URLs, text).
type ExtractionStage struct {
	extractor extractor.ContentExtractor
}

// NewExtractionStage creates a new ExtractionStage with the given extractor.
func NewExtractionStage(extractor extractor.ContentExtractor) *ExtractionStage {
	return &ExtractionStage{
		extractor: extractor,
	}
}

// Name returns "ExtractionStage".
func (s *ExtractionStage) Name() string {
	return "ExtractionStage"
}

// Execute extracts content from the source.
// Input: usecases.ContentSource
// Output: *PipelineData with ExtractionResult populated
func (s *ExtractionStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	source, ok := input.Data.(usecases.ContentSource)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for ExtractionStage: expected usecases.ContentSource, got %T", input.Data)
	}

	result, err := s.extractor.Extract(ctx, source)
	if err != nil {
		return StageOutput{}, err
	}

	data := &PipelineData{
		ExtractionResult: result,
	}

	return StageOutput{Data: data}, nil
}
