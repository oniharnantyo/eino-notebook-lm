package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// NewRetrievalAgent creates a new retrieval agent
func NewRetrievalAgent(ctx context.Context, model model.ToolCallingChatModel, tools []tool.BaseTool) (adk.Agent, error) {
	// Create the agent using Eino ADK
	config := &adk.ChatModelAgentConfig{
		Name:        "RetrievalAgent",
		Description: "An agent that can search for information in documents",
		Model:       model,
		Instruction: BaseAgentInstruction,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 10,
	}

	return adk.NewChatModelAgent(ctx, config)
}

// BuildCatalog constructs a string representation of available sources for agent prompting.
// It fetches source metadata from the repository for the given source IDs.
func BuildCatalog(ctx context.Context, sourceRepo repositories.SourceRepository, sourceIDs []uuid.UUID) (string, error) {
	if len(sourceIDs) == 0 {
		return "No sources available.", nil
	}

	sources, err := sourceRepo.ListSourceSummariesByID(ctx, sourceIDs)
	if err != nil {
		return "", fmt.Errorf("failed to fetch sources for catalog: %w", err)
	}

	if len(sources) == 0 {
		return "No sources available.", nil
	}

	var sb strings.Builder
	sb.WriteString("Available Sources:\n")
	for _, s := range sources {
		sb.WriteString(fmt.Sprintf("- ID: %s, Title: %s\n", s.ID.String(), s.Title))
	}
	return sb.String(), nil
}
