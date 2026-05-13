package agent

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

type RetrievalAgent struct {
	model       model.ToolCallingChatModel
	staticTools []tool.BaseTool
}

func NewRetrievalAgent(
	model model.ToolCallingChatModel,
	staticTools ...tool.BaseTool,
) *RetrievalAgent {
	return &RetrievalAgent{
		model:       model,
		staticTools: staticTools,
	}
}

func (a *RetrievalAgent) Invoke(ctx context.Context, extraTools ...tool.BaseTool) (adk.Agent, error) {
	allTools := make([]tool.BaseTool, len(a.staticTools), len(a.staticTools)+len(extraTools))
	copy(allTools, a.staticTools)
	allTools = append(allTools, extraTools...)

	config := &adk.ChatModelAgentConfig{
		Name:        "RetrievalAgent",
		Description: "An agent that can search for information in documents",
		Model:       a.model,
		Instruction: BaseAgentInstruction,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: allTools,
			},
		},
		MaxIterations: 20,
	}

	return adk.NewChatModelAgent(ctx, config)
}
