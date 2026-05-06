package stages

import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
)

// mapAgentEventToSSE converts ADK events to schema.Message for streaming.
func mapAgentEventToSSE(event *adk.AgentEvent) (*dtos.ResponseResource, *schema.Message, error) {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil, nil
	}
	msg, err := event.Output.MessageOutput.GetMessage()
	if err != nil {
		return nil, nil, err
	}
	return nil, msg, nil
}
