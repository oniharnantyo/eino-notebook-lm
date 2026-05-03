package pipeline

import (
	"context"
	"fmt"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

type StatusUpdateStage struct {
	sourceRepo repositories.SourceRepository
}

func NewStatusUpdateStage(sourceRepo repositories.SourceRepository) *StatusUpdateStage {
	return &StatusUpdateStage{
		sourceRepo: sourceRepo,
	}
}

func (s *StatusUpdateStage) Name() string { return "StatusUpdateStage" }

func (s *StatusUpdateStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	source, err := s.sourceRepo.GetByID(ctx, input.SourceID)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to get source: %w", err)
	}
	if source == nil {
		return StageOutput{}, fmt.Errorf("source not found: %s", input.SourceID)
	}

	source.MarkCompleted()

	if err := s.sourceRepo.Update(ctx, source); err != nil {
		return StageOutput{}, fmt.Errorf("failed to update source status: %w", err)
	}

	return StageOutput{Data: input.Data}, nil
}
