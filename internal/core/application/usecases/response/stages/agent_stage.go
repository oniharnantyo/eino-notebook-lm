package stages

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/agent"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/agent/tools"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// AgentStage handles the agent-based response generation
type AgentStage struct {
	chatModel   model.ToolCallingChatModel
	sourceRepo  repositories.SourceRepository
	toolFactory *tools.ToolFactory
}

// NewAgentStage creates a new AgentStage
func NewAgentStage(chatModel model.ToolCallingChatModel, sourceRepo repositories.SourceRepository, toolFactory *tools.ToolFactory) *AgentStage {
	return &AgentStage{
		chatModel:   chatModel,
		sourceRepo:  sourceRepo,
		toolFactory: toolFactory,
	}
}

// Execute runs the agent
func (s *AgentStage) Execute(ctx context.Context, input *schema.Message, sourceIDs []uuid.UUID, tools []any) (GenerationOutput, error) {
	// Build source catalog for agent grounding
	catalog, err := agent.BuildCatalog(ctx, s.sourceRepo, sourceIDs)
	if err != nil {
		return GenerationOutput{}, fmt.Errorf("failed to build source catalog: %w", err)
	}

	// Create agent instance
	// Need to cast []any to []tool.BaseTool
	var agentTools []tool.BaseTool
	for _, t := range tools {
		if bt, ok := t.(tool.BaseTool); ok {
			agentTools = append(agentTools, bt)
		}
	}

	ag, err := agent.NewRetrievalAgent(ctx, s.chatModel, agentTools)
	if err != nil {
		return GenerationOutput{}, fmt.Errorf("failed to create agent: %w", err)
	}

	// Create ADK Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           ag,
		EnableStreaming: true,
	})

	// Run agent
	// Pass catalog via session values - ADK will replace {catalog} in BaseAgentInstruction
	iter := runner.Query(ctx, input.Content, adk.WithSessionValues(map[string]any{"catalog": catalog}))

	// Use a pipe to create a stream reader
	pr, pw := schema.Pipe[*schema.Message](10)

	go func() {
		defer pw.Close()
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			_, msg, err := mapAgentEventToSSE(event)
			if err != nil {
				// Handle error
				break
			}
			if msg != nil {
				_ = pw.Send(msg, nil) // Send accepts (message, error)
			}
		}
	}()

	return GenerationOutput{Stream: pr}, nil
}
