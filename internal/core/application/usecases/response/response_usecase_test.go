package response

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/stages"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/models"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

type mockRetriever struct{ mock.Mock }

func (m *mockRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	args := m.Called(ctx, query, opts)
	return args.Get(0).([]*schema.Document), args.Error(1)
}

type mockEmbedder struct{ mock.Mock }

func (m *mockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	args := m.Called(ctx, texts, opts)
	return args.Get(0).([][]float64), args.Error(1)
}

// TestResponseUseCase_Stream_StreamingFlow tests the full streaming flow
func TestResponseUseCase_Stream_StreamingFlow(t *testing.T) {
	ctx := context.Background()
	nbRepo := &repositories.MockNotebookRepository{}
	cvRepo := &repositories.MockConversationRepository{}
	emb := &mockEmbedder{}
	cm := &models.MockToolCallingChatModel{}

	notebookID := uuid.New()
	notebookIDStr := notebookID.String()

	nbRepo.On("Exists", ctx, notebookID).Return(true, nil)

	historyConfig := &history.HistoryConfig{MaxMessages: 10}

	testChunks := []string{"Hello", " world", "!"}
	expectedText := "Hello world!"

	pr, pw := schema.Pipe[*schema.Message](10)

	mockAgent := new(mockAgentStage)
	mockHistory := new(mockHistoryStage)

	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)
	mockHistory.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockAgent.On("Execute", ctx, mock.Anything, mock.Anything).Return(stages.GenerationOutput{Stream: pr}, nil)

	uc := &responseUseCase{
		notebookRepo:     nbRepo,
		conversationRepo: cvRepo,
		sourceRepo:       nil,
		embedder:         emb,
		chatModel:        cm,
		defaultModel:     "test-model",
		historyManager:   history.NewHistoryManager(historyConfig),
		pipeline:         NewResponsePipeline(mockAgent, mockHistory),
	}

	req := &dtos.ResponseRequest{
		NotebookID: &notebookIDStr,
		Input:      "Test streaming",
		Stream:     true,
	}

	streamReader, meta, err := uc.Stream(ctx, req)
	assert.NoError(t, err, "Stream should not return an error")
	assert.NotNil(t, streamReader, "Stream reader should not be nil")
	assert.NotNil(t, meta, "Stream meta should not be nil")

	go func() {
		defer pw.Close()
		for _, chunk := range testChunks {
			_ = pw.Send(&schema.Message{Content: chunk}, nil)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	var accumulated strings.Builder
	for {
		msg, err := streamReader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Failed to receive from stream: %v", err)
		}
		if msg != nil {
			accumulated.WriteString(msg.Content)
		}
	}

	assert.Equal(t, expectedText, accumulated.String(), "Accumulated text should match expected output")

	streamReader.Close()

	nbRepo.AssertExpectations(t)
	mockHistory.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
}

// TestResponseUseCase_Stream_WithToolCalls tests streaming flow when the agent makes tool calls
func TestResponseUseCase_Stream_WithToolCalls(t *testing.T) {
	ctx := context.Background()
	nbRepo := &repositories.MockNotebookRepository{}
	cvRepo := &repositories.MockConversationRepository{}
	emb := &mockEmbedder{}
	cm := &models.MockToolCallingChatModel{}

	notebookID := uuid.New()
	notebookIDStr := notebookID.String()

	nbRepo.On("Exists", ctx, notebookID).Return(true, nil)

	historyConfig := &history.HistoryConfig{MaxMessages: 10}

	testChunks := []string{"Searching", " for", " relevant", " documents", "..."}

	pr, pw := schema.Pipe[*schema.Message](10)

	mockAgent := new(mockAgentStage)
	mockHistory := new(mockHistoryStage)

	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)
	mockHistory.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockAgent.On("Execute", ctx, mock.Anything, mock.Anything).Return(stages.GenerationOutput{Stream: pr}, nil)

	uc := &responseUseCase{
		notebookRepo:     nbRepo,
		conversationRepo: cvRepo,
		sourceRepo:       nil,
		embedder:         emb,
		chatModel:        cm,
		defaultModel:     "test-model",
		historyManager:   history.NewHistoryManager(historyConfig),
		pipeline:         NewResponsePipeline(mockAgent, mockHistory),
	}

	req := &dtos.ResponseRequest{
		NotebookID: &notebookIDStr,
		Input:      "Search for information about semantic search",
		Stream:     true,
	}

	streamReader, meta, err := uc.Stream(ctx, req)
	assert.NoError(t, err, "Stream should not return an error")
	assert.NotNil(t, streamReader, "Stream reader should not be nil")
	assert.NotNil(t, meta, "Stream meta should not be nil")

	go func() {
		defer pw.Close()
		for _, chunk := range testChunks {
			msg := &schema.Message{
				Role:    schema.Assistant,
				Content: chunk,
			}
			_ = pw.Send(msg, nil)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	var accumulated strings.Builder
	for {
		msg, err := streamReader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Failed to receive from stream: %v", err)
		}
		if msg != nil {
			accumulated.WriteString(msg.Content)
		}
	}

	expectedText := strings.Join(testChunks, "")
	assert.Equal(t, expectedText, accumulated.String(), "Accumulated text should match expected output")

	streamReader.Close()

	nbRepo.AssertExpectations(t)
	mockHistory.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
}

// TestResponseUseCase_ValidateNotebook tests notebook validation
func TestResponseUseCase_ValidateNotebook(t *testing.T) {
	ctx := context.Background()
	nbRepo := &repositories.MockNotebookRepository{}

	uc := &responseUseCase{
		notebookRepo: nbRepo,
	}

	t.Run("valid_notebook", func(t *testing.T) {
		notebookID := uuid.New()
		notebookIDStr := notebookID.String()

		nbRepo.On("Exists", ctx, notebookID).Return(true, nil)

		req := &dtos.ResponseRequest{
			NotebookID: &notebookIDStr,
		}

		result, err := uc.validateNotebook(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, &notebookID, result)
		nbRepo.AssertExpectations(t)
	})

	t.Run("missing_notebook_id", func(t *testing.T) {
		req := &dtos.ResponseRequest{
			Input: "test",
		}

		result, err := uc.validateNotebook(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "notebook id is required")
	})

	t.Run("invalid_uuid", func(t *testing.T) {
		invalidID := "not-a-uuid"
		req := &dtos.ResponseRequest{
			NotebookID: &invalidID,
		}

		result, err := uc.validateNotebook(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid notebook_id")
	})

	t.Run("notebook_not_found", func(t *testing.T) {
		notebookID := uuid.New()
		notebookIDStr := notebookID.String()

		nbRepo.On("Exists", ctx, notebookID).Return(false, nil)

		req := &dtos.ResponseRequest{
			NotebookID: &notebookIDStr,
		}

		result, err := uc.validateNotebook(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "notebook not found")
		nbRepo.AssertExpectations(t)
	})
}
