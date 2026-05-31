package stages

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/stretchr/testify/assert"
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
