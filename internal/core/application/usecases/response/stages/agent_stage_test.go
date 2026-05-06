package stages

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/models"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAgentStage_Execute(t *testing.T) {
	mockModel := new(models.MockToolCallingChatModel)
	mockRepo := new(repositories.MockSourceRepository)

	// Create a pipe to simulate the stream
	pr, pw := schema.Pipe[*schema.Message](1)

	// Prepare expectations
	mockModel.On("Stream", mock.Anything, mock.Anything, mock.Anything).Return(pr, nil)
	mockRepo.On("ListSourceSummariesByID", mock.Anything, mock.Anything).Return([]*entities.Source{}, nil)

	stage := NewAgentStage(mockModel, mockRepo, nil)

	input := &schema.Message{
		Role:    schema.User,
		Content: "Search for something",
	}

	// Execute should return a non-nil stream
	output, err := stage.Execute(context.Background(), input, []uuid.UUID{}, []any{})

	assert.NoError(t, err)
	assert.NotNil(t, output.Stream)

	// Close the writer and check for io.EOF
	go func() {
		pw.Close()
	}()

	// Drain the stream until EOF
	for {
		_, err := output.Stream.Recv()
		if err != nil {
			assert.Equal(t, io.EOF, err)
			break
		}
	}
}
