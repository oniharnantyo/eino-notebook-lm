package stages

import (
	"context"
	"io"
	"testing"

	"github.com/cloudwego/eino/schema"
	agent "github.com/oniharnantyo/eino-notebook/internal/core/application/agent/retrieval"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/models"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAgentStage_Execute(t *testing.T) {
	mockModel := new(models.MockToolCallingChatModel)
	mockSourceRepo := new(repositories.MockSourceRepository)
	mockKnowledgeRepo := new(repositories.MockKnowledgeRepository)

	retrievalAgent := agent.NewRetrievalAgent(mockModel)

	// Create a pipe to simulate the stream
	pr, pw := schema.Pipe[*schema.Message](1)

	// Prepare expectations
	mockModel.On("Stream", mock.Anything, mock.Anything, mock.Anything).Return(pr, nil)
	mockSourceRepo.On("ListSourceSummariesByID", mock.Anything, mock.Anything).Return([]*entities.Source{}, nil)

	stage := NewAgentStage(retrievalAgent, mockSourceRepo, mockKnowledgeRepo)

	input := &schema.Message{
		Role:    schema.User,
		Content: "Search for something",
	}

	// Execute should return a non-nil stream
	output, err := stage.Execute(context.Background(), input, []uuid.UUID{})

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

func TestAgentStage_Execute_WithContent(t *testing.T) {
	mockModel := new(models.MockToolCallingChatModel)
	mockSourceRepo := new(repositories.MockSourceRepository)
	mockKnowledgeRepo := new(repositories.MockKnowledgeRepository)

	retrievalAgent := agent.NewRetrievalAgent(mockModel)

	// Create a pipe to simulate the stream
	pr, pw := schema.Pipe[*schema.Message](1)

	// Prepare expectations
	mockModel.On("Stream", mock.Anything, mock.Anything, mock.Anything).Return(pr, nil)
	mockSourceRepo.On("ListSourceSummariesByID", mock.Anything, mock.Anything).Return([]*entities.Source{}, nil)

	stage := NewAgentStage(retrievalAgent, mockSourceRepo, mockKnowledgeRepo)

	input := &schema.Message{
		Role:    schema.User,
		Content: "Search for something",
	}

	// Execute should return a non-nil stream
	output, err := stage.Execute(context.Background(), input, []uuid.UUID{})

	assert.NoError(t, err)
	assert.NotNil(t, output.Stream)

	// Simulate model output
	go func() {
		pw.Send(&schema.Message{
			Role:    schema.Assistant,
			Content: "Test response",
		}, nil)
		pw.Close()
	}()

	// Verify content is received
	msg, err := output.Stream.Recv()
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, "Test response", msg.Content)

	// Next recv should return EOF
	_, err = output.Stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestAgentStage_ThinkingParser(t *testing.T) {
	tests := []struct {
		name              string
		chunks            []string
		expectedReasoning string
		expectedContent   string
	}{
		{
			name:              "Simple single chunk",
			chunks:            []string{"<thinking>thinking</thinking>hello"},
			expectedReasoning: "thinking",
			expectedContent:   "hello",
		},
		{
			name:              "Split tags",
			chunks:            []string{"<th", "ink>thinking</thi", "nk>hello"},
			expectedReasoning: "thinking",
			expectedContent:   "hello",
		},
		{
			name:              "Mixed content",
			chunks:            []string{"Start <thinking>thinking more</thinking> End"},
			expectedReasoning: "thinking more",
			expectedContent:   "Start  End",
		},
		{
			name:              "No tags",
			chunks:            []string{"Just", " content"},
			expectedReasoning: "",
			expectedContent:   "Just content",
		},
		{
			name:              "Only reasoning",
			chunks:            []string{"<thinking>only reasoning</thinking>"},
			expectedReasoning: "only reasoning",
			expectedContent:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &thinkingParser{}
			var gotReasoning, gotContent string
			for _, chunk := range tt.chunks {
				r, c := p.Process(chunk)
				gotReasoning += r
				gotContent += c
			}
			assert.Equal(t, tt.expectedReasoning, gotReasoning)
			assert.Equal(t, tt.expectedContent, gotContent)
		})
	}
}
