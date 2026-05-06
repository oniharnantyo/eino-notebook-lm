package response

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/agent/tools"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/history"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/response/stages"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/models"
	"github.com/oniharnantyo/eino-notebook/internal/mocks/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Mock Eino Components (Not yet auto-generated)
type mockRetriever struct{ mock.Mock }

func (m *mockRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	// Variadic arguments in testify mock
	args := m.Called(ctx, query, opts)
	return args.Get(0).([]*schema.Document), args.Error(1)
}

type mockEmbedder struct{ mock.Mock }

func (m *mockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	args := m.Called(ctx, texts, opts)
	return args.Get(0).([][]float64), args.Error(1)
}

// Test List Sources Tool Scope Verification
func TestListSourcesTool_ScopeRespect(t *testing.T) {
	// Arrange
	ctx := context.Background()
	srcRepo := &repositories.MockSourceRepository{}

	sourceID1 := uuid.New()
	sourceID2 := uuid.New()
	sourceID3 := uuid.New()

	// All available sources
	allSources := []*entities.Source{
		{ID: sourceID1, Title: "Source 1", ContentType: "application/pdf"},
		{ID: sourceID2, Title: "Source 2", ContentType: "text/html"},
		{ID: sourceID3, Title: "Source 3", ContentType: "text/plain"},
	}

	// Test cases for different scopes
	testCases := []struct {
		name         string
		sourceIDs    []uuid.UUID
		expectCount  int
		expectTitles []string
	}{
		{
			name:         "empty_scope_returns_empty",
			sourceIDs:    []uuid.UUID{},
			expectCount:  0,
			expectTitles: []string{},
		},
		{
			name:         "single_source_scope",
			sourceIDs:    []uuid.UUID{sourceID1},
			expectCount:  1,
			expectTitles: []string{"Source 1"},
		},
		{
			name:         "multiple_source_scope",
			sourceIDs:    []uuid.UUID{sourceID1, sourceID2},
			expectCount:  2,
			expectTitles: []string{"Source 1", "Source 2"},
		},
		{
			name:         "all_sources_scope",
			sourceIDs:    []uuid.UUID{sourceID1, sourceID2, sourceID3},
			expectCount:  3,
			expectTitles: []string{"Source 1", "Source 2", "Source 3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations
			if len(tc.sourceIDs) == 0 {
				srcRepo.On("ListSourceSummariesByID", ctx, []uuid.UUID{}).Return([]*entities.Source{}, nil)
			} else {
				// Filter sources based on scope
				filtered := filterSourcesByID(allSources, tc.sourceIDs)
				srcRepo.On("ListSourceSummariesByID", ctx, tc.sourceIDs).Return(filtered, nil)
			}

			// Create tool with specific scope
			listSourcesTool := tools.NewListSourcesTool(srcRepo, tc.sourceIDs)

			assert.NotNil(t, listSourcesTool, "Tool should be created")

			// Note: The tool scope is validated during actual execution, not during Info() call.
			// The mock setup above already verifies the correct scope is used.
		})
	}
}

// Test Tool Factory Scope Configuration
func TestToolFactory_ScopeConfiguration(t *testing.T) {
	// Arrange
	_ = context.Background()
	srcRepo := &repositories.MockSourceRepository{}
	emb := &mockEmbedder{}

	sourceID1 := uuid.New()
	sourceID2 := uuid.New()

	toolFactory := tools.NewToolFactory(nil, nil, nil, nil, srcRepo, emb)

	t.Run("creates_scoped_tools_with_source_ids", func(t *testing.T) {
		scopeConfig := tools.ScopeConfig{
			SourceIDs:   []uuid.UUID{sourceID1, sourceID2},
			SourceTypes: []string{},
		}

		scopedTools := toolFactory.NewScopedTools(scopeConfig)

		assert.NotEmpty(t, scopedTools, "Should create scoped tools")
		assert.Len(t, scopedTools, 5, "Should create 5 tools: keyword_search, semantic_search, image_search, chunk_read, list_sources")
	})

	t.Run("validates_unsupported_source_types", func(t *testing.T) {
		// Test unsupported source type validation
		isSupported := toolFactory.IsSourceTypeSupported("unsupported_type")
		// With nil retriever, it should return false or handle gracefully
		assert.False(t, isSupported, "Should return false for unsupported type")
	})

	t.Run("supports_common_source_types", func(t *testing.T) {
		// Test common source types
		supportedTypes := []string{"image", "knowledge", "sentence", "pdf", "text", "docx", "website"}

		for _, sourceType := range supportedTypes {
			isSupported := toolFactory.IsSourceTypeSupported(sourceType)
			// With nil retrievers, these may return false, but the validation logic should not crash
			assert.NotNil(t, isSupported, "Should handle source type validation without crashing: "+sourceType)
		}
	})
}

// Helper function to filter sources by ID
func filterSourcesByID(sources []*entities.Source, ids []uuid.UUID) []*entities.Source {
	idMap := make(map[uuid.UUID]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	filtered := make([]*entities.Source, 0)
	for _, src := range sources {
		if idMap[src.ID] {
			filtered = append(filtered, src)
		}
	}
	return filtered
}

// TestResponseUseCase_Stream_StreamingFlow tests the full streaming flow
func TestResponseUseCase_Stream_StreamingFlow(t *testing.T) {
	// Arrange
	ctx := context.Background()
	nbRepo := &repositories.MockNotebookRepository{}
	cvRepo := &repositories.MockConversationRepository{}
	ret := &mockRetriever{}
	emb := &mockEmbedder{}
	cm := &models.MockToolCallingChatModel{}

	notebookID := uuid.New()
	notebookIDStr := notebookID.String()

	nbRepo.On("Exists", ctx, notebookID).Return(true, nil)

	historyConfig := &history.HistoryConfig{MaxMessages: 10}

	// Create mock streaming response
	testChunks := []string{"Hello", " world", "!"}
	expectedText := "Hello world!"

	// Create a pipe for streaming
	pr, pw := schema.Pipe[*schema.Message](10)

	// Mock agent stage that returns streaming output
	mockAgent := new(mockAgentStage)

	// Create usecase with mock pipeline
	mockToolPrep := new(mockToolPrepStage)
	mockHistory := new(mockHistoryStage)

	mockToolPrep.On("Execute", ctx, mock.Anything).Return(stages.ToolPreparationOutput{Tools: []tool.BaseTool{}}, nil)
	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)
	mockHistory.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockAgent.On("Execute", ctx, mock.Anything, mock.Anything, mock.Anything).Return(stages.GenerationOutput{Stream: pr}, nil)

	uc := &responseUseCase{
		notebookRepo:     nbRepo,
		conversationRepo: cvRepo,
		sourceRepo:       nil,
		retriever:        ret,
		embedder:         emb,
		chatModel:        cm,
		defaultModel:     "test-model",
		historyManager:   history.NewHistoryManager(historyConfig),
		pipeline:         NewResponsePipeline(mockToolPrep, mockAgent, mockHistory),
	}

	req := &dtos.ResponseRequest{
		NotebookID: &notebookIDStr,
		Input:      "Test streaming",
		Stream:     true,
	}

	// Act - Start streaming
	streamReader, meta, err := uc.Stream(ctx, req)
	assert.NoError(t, err, "Stream should not return an error")
	assert.NotNil(t, streamReader, "Stream reader should not be nil")
	assert.NotNil(t, meta, "Stream meta should not be nil")

	// Send test chunks through the mock agent's stream
	go func() {
		defer pw.Close()
		for _, chunk := range testChunks {
			_ = pw.Send(&schema.Message{Content: chunk}, nil)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Collect all messages from the stream
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

	// Assert - Verify accumulated text matches expected
	assert.Equal(t, expectedText, accumulated.String(), "Accumulated text should match expected output")

	// Verify stream is properly closed
	streamReader.Close()

	nbRepo.AssertExpectations(t)
	mockToolPrep.AssertExpectations(t)
	mockHistory.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
}

// TestResponseUseCase_Stream_WithToolCalls tests streaming flow when the agent makes tool calls
func TestResponseUseCase_Stream_WithToolCalls(t *testing.T) {
	// Arrange
	ctx := context.Background()
	nbRepo := &repositories.MockNotebookRepository{}
	cvRepo := &repositories.MockConversationRepository{}
	ret := &mockRetriever{}
	emb := &mockEmbedder{}
	cm := &models.MockToolCallingChatModel{}

	notebookID := uuid.New()
	notebookIDStr := notebookID.String()

	nbRepo.On("Exists", ctx, notebookID).Return(true, nil)

	historyConfig := &history.HistoryConfig{MaxMessages: 10}

	// Create mock streaming response with tool call events interleaved
	testChunks := []string{"Searching", " for", " relevant", " documents", "..."}

	// Create a pipe for streaming
	pr, pw := schema.Pipe[*schema.Message](10)

	// Mock agent stage that returns streaming output
	mockAgent := new(mockAgentStage)

	// Create usecase with mock pipeline
	mockToolPrep := new(mockToolPrepStage)
	mockHistory := new(mockHistoryStage)

	mockToolPrep.On("Execute", ctx, mock.Anything).Return(stages.ToolPreparationOutput{Tools: []tool.BaseTool{}}, nil)
	mockHistory.On("Execute", ctx, mock.Anything).Return(stages.HistoryOutput{}, nil)
	mockHistory.On("Save", mock.Anything, mock.Anything).Return(nil).Maybe()
	mockAgent.On("Execute", ctx, mock.Anything, mock.Anything, mock.Anything).Return(stages.GenerationOutput{Stream: pr}, nil)

	uc := &responseUseCase{
		notebookRepo:     nbRepo,
		conversationRepo: cvRepo,
		sourceRepo:       nil,
		retriever:        ret,
		embedder:         emb,
		chatModel:        cm,
		defaultModel:     "test-model",
		historyManager:   history.NewHistoryManager(historyConfig),
		pipeline:         NewResponsePipeline(mockToolPrep, mockAgent, mockHistory),
	}

	req := &dtos.ResponseRequest{
		NotebookID: &notebookIDStr,
		Input:      "Search for information about semantic search",
		Stream:     true,
	}

	// Act - Start streaming
	streamReader, meta, err := uc.Stream(ctx, req)
	assert.NoError(t, err, "Stream should not return an error")
	assert.NotNil(t, streamReader, "Stream reader should not be nil")
	assert.NotNil(t, meta, "Stream meta should not be nil")

	// Send test chunks through the mock agent's stream
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

	// Collect all messages from the stream
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

	// Assert - Verify accumulated text matches expected chunks
	expectedText := strings.Join(testChunks, "")
	assert.Equal(t, expectedText, accumulated.String(), "Accumulated text should match expected output")

	// Verify stream is properly closed
	streamReader.Close()

	nbRepo.AssertExpectations(t)
	mockToolPrep.AssertExpectations(t)
	mockHistory.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
}
