package stages

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHistoryStage_Load(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := &repositories.MockConversationRepository{}
	hm := history.NewHistoryManager(&history.HistoryConfig{
		Strategy:    history.HistoryStrategySlidingWindow,
		MaxMessages: 10,
	})
	stage := NewHistoryStage(hm, repo)

	prevID := "prev_123"
	repo.On("FindByResponseID", ctx, prevID).Return(&entities.Conversation{
		ResponseID: prevID,
		Messages: []*entities.StoredMessage{
			{Role: "user", Content: "Hi"},
			{Role: "assistant", Content: "Hello world"},
		},
	}, nil)

	input := HistoryInput{
		PreviousResponseID: &prevID,
	}

	// Act
	output, err := stage.Load(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, output.Messages, 2)
	assert.Equal(t, "Hi", output.Messages[0].Content)
	repo.AssertExpectations(t)
}

func TestHistoryStage_Save(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := &repositories.MockConversationRepository{}
	hm := history.NewHistoryManager(nil)
	stage := NewHistoryStage(hm, repo)

	notebookID := uuid.New().String()
	responseID := "resp_456"

	repo.On("Save", ctx, mock.MatchedBy(func(c *entities.Conversation) bool {
		return c.ResponseID == responseID && *c.NotebookID == notebookID
	})).Return(nil)

	input := HistorySaveInput{
		NotebookID: notebookID,
		ResponseID: responseID,
		History: []*schema.Message{
			{Role: schema.User, Content: "Old msg"},
		},
		UserInput: "New msg",
		ResponseMessage: &schema.Message{
			Role:    schema.Assistant,
			Content: "Response msg",
		},
		Model: "gpt-4",
	}

	// Act
	err := stage.Save(ctx, input)

	// Assert
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
