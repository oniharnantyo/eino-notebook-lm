package stages

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHistoryStage_Load(t *testing.T) {
	// Arrange
	ctx := context.Background()
	repo := new(repositories.MockConversationRepository)
	hm := history.NewHistoryManager(&history.HistoryConfig{
		Strategy:    history.HistoryStrategySlidingWindow,
		MaxMessages: 10,
	})
	stage := NewHistoryStage(hm, repo)

	prevID := "prev_123"
	conv := &entities.Conversation{
		ID: "conv_123",
	}

	repo.On("FindByResponseID", ctx, prevID).Return(conv, nil)

	messages := []*entities.Message{
		{
			Message: &entities.StoredMessage{Role: "assistant", Content: "Hello world"},
		},
		{
			Message: &entities.StoredMessage{Role: "user", Content: "Hi"},
		},
	}
	repo.On("GetMessages", ctx, conv.ID, 100, mock.AnythingOfType("*int"), mock.Anything).Return(messages, nil)

	input := HistoryInput{
		PreviousResponseID: &prevID,
	}

	// Act
	output, err := stage.Load(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, output.Messages, 2)
	// Since GetMessages returns DESC, the loop in Load reverses it back to ASC
	assert.Equal(t, "user", string(output.Messages[0].Role))
	assert.Equal(t, "Hi", output.Messages[0].Content)
	repo.AssertExpectations(t)
}

