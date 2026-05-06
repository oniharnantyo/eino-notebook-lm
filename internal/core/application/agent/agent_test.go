package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/models"
	"github.com/stretchr/testify/assert"
)

func TestNewRetrievalAgent(t *testing.T) {
	ctx := context.Background()
	mockModel := new(models.MockToolCallingChatModel)
	tools := []tool.BaseTool{}

	agent, err := NewRetrievalAgent(ctx, mockModel, tools)
	assert.NoError(t, err)
	assert.NotNil(t, agent)
}
