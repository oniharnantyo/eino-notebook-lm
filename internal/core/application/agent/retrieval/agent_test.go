package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/models"
	"github.com/stretchr/testify/assert"
)

func TestNewRetrievalAgent(t *testing.T) {
	mockModel := new(models.MockToolCallingChatModel)
	staticTools := []tool.BaseTool{}

	agent := NewRetrievalAgent(mockModel, staticTools...)
	assert.NotNil(t, agent)
	assert.Equal(t, mockModel, agent.model)
	assert.Len(t, agent.staticTools, 0)
}

func TestRetrievalAgent_Invoke(t *testing.T) {
	ctx := context.Background()
	mockModel := new(models.MockToolCallingChatModel)

	agent := NewRetrievalAgent(mockModel)

	// Invoke should create an ADK agent even with no tools
	adkAgent, err := agent.Invoke(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, adkAgent)
}
