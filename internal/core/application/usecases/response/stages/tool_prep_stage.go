package stages

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/agent/tools"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type ToolPreparationStage struct {
	toolFactory *tools.ToolFactory
}

func NewToolPreparationStage(f *tools.ToolFactory) *ToolPreparationStage {
	return &ToolPreparationStage{toolFactory: f}
}

func (s *ToolPreparationStage) Execute(ctx context.Context, input ToolPreparationInput) (ToolPreparationOutput, error) {
	sourceIDs := make([]uuid.UUID, 0, len(input.SourceIDs))
	for _, id := range input.SourceIDs {
		u, err := uuid.Parse(id)
		if err == nil {
			sourceIDs = append(sourceIDs, u)
		}
	}

	// Validate SourceTypes
	for _, st := range input.SourceTypes {
		if !s.toolFactory.IsSourceTypeSupported(st) {
			return ToolPreparationOutput{}, fmt.Errorf("unsupported source type: %s", st)
		}
	}

	cfg := tools.ScopeConfig{
		SourceIDs:   sourceIDs,
		SourceTypes: input.SourceTypes,
	}

	return ToolPreparationOutput{
		Tools: s.toolFactory.NewScopedTools(cfg),
	}, nil
}
