package response

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/stages"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type mockAgentStage struct {
	mock.Mock
}

func (m *mockAgentStage) Execute(ctx context.Context, input *schema.Message, sourceIDs []uuid.UUID) (stages.GenerationOutput, error) {
	args := m.Called(ctx, input, sourceIDs)
	return args.Get(0).(stages.GenerationOutput), args.Error(1)
}

type mockHistoryStage struct {
	mock.Mock
}

func (m *mockHistoryStage) Execute(ctx context.Context, input stages.HistoryInput) (stages.HistoryOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(stages.HistoryOutput), args.Error(1)
}

func (m *mockHistoryStage) Save(ctx context.Context, input stages.HistorySaveInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func TestResponsePipeline_Execute_Success(t *testing.T) {
	mockAgent := new(mockAgentStage)
	mockHistory := new(mockHistoryStage)

	pipeline := NewResponsePipeline(mockAgent, mockHistory)

	ctx := context.Background()
	notebookID := "nb1"
	req := &dtos.ResponseRequest{
		NotebookID: &notebookID,
		Input:      "hello",
	}

	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)
	mockAgent.On("Execute", ctx, mock.Anything, mock.Anything).Return(stages.GenerationOutput{
		Stream: nil,
	}, nil)
	mockHistory.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()

	_, _, err := pipeline.Execute(ctx, req, "system", "model")

	assert.NoError(t, err)
	mockHistory.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
}

func TestResponsePipeline_Execute_HistoryFailure(t *testing.T) {
	mockAgent := new(mockAgentStage)
	mockHistory := new(mockHistoryStage)

	pipeline := NewResponsePipeline(mockAgent, mockHistory)

	ctx := context.Background()
	req := &dtos.ResponseRequest{Input: "hello"}

	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, errors.New("history error"))

	_, _, err := pipeline.Execute(ctx, req, "system", "model")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "history stage failed")
}

func TestResponsePipeline_Execute_GenerationFailure(t *testing.T) {
	mockAgent := new(mockAgentStage)
	mockHistory := new(mockHistoryStage)

	pipeline := NewResponsePipeline(mockAgent, mockHistory)

	ctx := context.Background()
	req := &dtos.ResponseRequest{Input: "hello"}

	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)
	mockAgent.On("Execute", ctx, mock.Anything, mock.Anything).Return(stages.GenerationOutput{}, errors.New("agent error"))

	_, _, err := pipeline.Execute(ctx, req, "system", "model")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent stage failed")
}

func TestResponsePipeline_Execute_InvalidSourceID(t *testing.T) {
	mockAgent := new(mockAgentStage)
	mockHistory := new(mockHistoryStage)

	pipeline := NewResponsePipeline(mockAgent, mockHistory)

	ctx := context.Background()
	notebookID := "nb1"
	req := &dtos.ResponseRequest{
		NotebookID: &notebookID,
		Input:      "hello",
		SourceIDs:  []string{"invalid-uuid"},
	}

	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)

	_, _, err := pipeline.Execute(ctx, req, "system", "model")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source ID")
}
