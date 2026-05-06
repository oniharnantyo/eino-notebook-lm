package response

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/stages"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type ToolPreparationStage interface {
	Execute(ctx context.Context, input stages.ToolPreparationInput) (stages.ToolPreparationOutput, error)
}

type AgentStage interface {
	Execute(ctx context.Context, input *schema.Message, sourceIDs []uuid.UUID, tools []any) (stages.GenerationOutput, error)
}

type HistoryStage interface {
	Execute(ctx context.Context, input stages.HistoryInput) (stages.HistoryOutput, error)
	Save(ctx context.Context, input stages.HistorySaveInput) error
}

type ResponsePipeline struct {
	toolPrepStage   ToolPreparationStage
	agentStage      AgentStage
	historyStage    HistoryStage
}

func NewResponsePipeline(
	toolPrepStage ToolPreparationStage,
	agentStage AgentStage,
	historyStage HistoryStage,
) *ResponsePipeline {
	return &ResponsePipeline{
		toolPrepStage:   toolPrepStage,
		agentStage:      agentStage,
		historyStage:    historyStage,
	}
}

func (p *ResponsePipeline) Execute(ctx context.Context, req *dtos.ResponseRequest, systemPrompt, modelName string) (stages.GenerationOutput, []*schema.Message, error) {
	// Extract Input as string
	var userInput string
	if req.Input != nil {
		if str, ok := req.Input.(string); ok {
			userInput = str
		} else {
			userInput = fmt.Sprintf("%v", req.Input)
		}
	}

	notebookID := ""
	if req.NotebookID != nil {
		notebookID = *req.NotebookID
	}

	// 1. History (Load)
	histInput := stages.HistoryInput{
		NotebookID:         notebookID,
		PreviousResponseID: req.PreviousResponseID,
	}
	histOut, err := p.historyStage.Execute(ctx, histInput)
	if err != nil {
		return stages.GenerationOutput{}, nil, fmt.Errorf("history stage failed: %w", err)
	}

	// 2. Tool Prep
	toolInput := stages.ToolPreparationInput{
		SourceIDs:   req.SourceIDs,
		SourceTypes: req.SourceTypes,
	}
	toolOut, err := p.toolPrepStage.Execute(ctx, toolInput)
	if err != nil {
		return stages.GenerationOutput{}, nil, fmt.Errorf("tool prep stage failed: %w", err)
	}

	// 3. Generation
	tools := make([]any, len(toolOut.Tools))
	for i, t := range toolOut.Tools {
		tools[i] = t
	}

	sourceUUIDs := make([]uuid.UUID, 0, len(req.SourceIDs))
	for _, idStr := range req.SourceIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return stages.GenerationOutput{}, nil, fmt.Errorf("invalid source ID in pipeline: %s", idStr)
		}
		sourceUUIDs = append(sourceUUIDs, id)
	}

	msg := &schema.Message{Role: schema.User, Content: userInput}
	out, err := p.agentStage.Execute(ctx, msg, sourceUUIDs, tools)
	if err != nil {
		return stages.GenerationOutput{}, nil, fmt.Errorf("agent stage failed: %w", err)
	}

	return out, histOut.Messages, nil
}

