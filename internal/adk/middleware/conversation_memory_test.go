package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// mockConversationRepository is a test double for ConversationRepository
type mockConversationRepository struct {
	findByResponseIDFunc func(ctx context.Context, responseID string) (*entities.Conversation, error)
	findByIDFunc         func(ctx context.Context, id string) (*entities.Conversation, error)
	saveFunc             func(ctx context.Context, conversation *entities.Conversation, messages []*entities.Message) error
	getMessagesFunc      func(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error)
}

func (m *mockConversationRepository) Save(ctx context.Context, conversation *entities.Conversation, messages []*entities.Message) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, conversation, messages)
	}
	return nil
}

func (m *mockConversationRepository) GetMessages(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error) {
	if m.getMessagesFunc != nil {
		return m.getMessagesFunc(ctx, conversationID, limit, beforeSequence, isConversationHistory)
	}
	return nil, nil
}

func (m *mockConversationRepository) GetLatestConversationID(ctx context.Context, notebookID string) (string, error) {
	return "", nil
}

func (m *mockConversationRepository) FindByResponseID(ctx context.Context, responseID string) (*entities.Conversation, error) {
	if m.findByResponseIDFunc != nil {
		return m.findByResponseIDFunc(ctx, responseID)
	}
	return nil, nil
}

func (m *mockConversationRepository) FindByID(ctx context.Context, id string) (*entities.Conversation, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockConversationRepository) Delete(ctx context.Context, responseID string) error {
	return nil
}

func (m *mockConversationRepository) Exists(ctx context.Context, responseID string) (bool, error) {
	return false, nil
}

func (m *mockConversationRepository) List(ctx context.Context, filter repositories.ConversationFilter) ([]*entities.Conversation, int, error) {
	return nil, 0, nil
}

// TestBeforeAgent tests that BeforeAgent is a no-op
func TestBeforeAgent(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	ctx := context.Background()
	runCtx := &adk.ChatModelAgentContext{}

	newCtx, newRunCtx, err := middleware.BeforeAgent(ctx, runCtx)

	assert.NoError(t, err)
	assert.NotEqual(t, ctx, newCtx)
	assert.NotEmpty(t, newCtx.Value(runIDKey))
	assert.Equal(t, runCtx, newRunCtx)
}

// TestBeforeModelRewriteState tests loading conversation history
func TestBeforeModelRewriteState(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	tests := []struct {
		name              string
		previousResponseID interface{}
		mockConversation  *entities.Conversation
		mockMessages      []*entities.Message
		mockError         error
		expectMessages    int
		expectError       bool
	}{
		{
			name:              "No previous response ID",
			previousResponseID: "",
			mockConversation:  nil,
			mockMessages:      nil,
			mockError:         nil,
			expectMessages:    0,
			expectError:       false,
		},
		{
			name:              "Previous response ID not a string",
			previousResponseID: 123,
			mockConversation:  nil,
			mockMessages:      nil,
			mockError:         nil,
			expectMessages:    0,
			expectError:       false,
		},
		{
			name:              "Load conversation successfully",
			previousResponseID: "resp-123",
			mockConversation: &entities.Conversation{
				ID: "conv-123",
			},
			mockMessages: []*entities.Message{
				{
					Message: &entities.StoredMessage{Role: "assistant", Content: "Hi there"},
				},
				{
					Message: &entities.StoredMessage{Role: "user", Content: "Hello"},
				},
			},
			mockError:     nil,
			expectMessages: 2,
			expectError:    false,
		},
		{
			name:              "Load conversation returns nil",
			previousResponseID: "resp-404",
			mockConversation:  nil,
			mockMessages:      nil,
			mockError:         nil,
			expectMessages:    0,
			expectError:       false,
		},
		{
			name:              "Load conversation fails",
			previousResponseID: "resp-error",
			mockConversation:  nil,
			mockMessages:      nil,
			mockError:         errors.New("database error"),
			expectMessages:    0,
			expectError:       false, // Should gracefully degrade
		},
		{
			name:              "Load conversation with empty messages",
			previousResponseID: "resp-empty",
			mockConversation: &entities.Conversation{
				ID: "conv-empty",
			},
			mockMessages:  []*entities.Message{},
			mockError:     nil,
			expectMessages: 0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			repo.findByResponseIDFunc = func(ctx context.Context, responseID string) (*entities.Conversation, error) {
				return tt.mockConversation, tt.mockError
			}
			repo.getMessagesFunc = func(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error) {
				return tt.mockMessages, nil
			}

			// Create context with previous_response_id
			ctx := context.Background()
			if tt.previousResponseID != nil {
				ctx = context.WithValue(ctx, "previous_response_id", tt.previousResponseID)
			}

			state := &adk.ChatModelAgentState{
				Messages: []*schema.Message{},
			}

			newCtx, newState, err := middleware.BeforeModelRewriteState(ctx, state, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check that original context values are preserved
			if tt.previousResponseID != nil {
				assert.Equal(t, tt.previousResponseID, newCtx.Value("previous_response_id"))
			}
			// Check that history_message_count is set when messages are loaded
			if tt.expectMessages > 0 {
				historyCount, ok := newCtx.Value("history_message_count").(int)
				assert.True(t, ok, "history_message_count should be set in context")
				assert.Equal(t, tt.expectMessages, historyCount)
			}
			assert.NotNil(t, newState)

			if tt.expectMessages > 0 {
				assert.Len(t, newState.Messages, tt.expectMessages)
			} else {
				assert.Len(t, newState.Messages, 0)
			}
		})
	}
}

// TestAfterModelRewriteState tests that AfterModelRewriteState is a no-op
func TestAfterModelRewriteState(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	ctx := context.Background()
	state := &adk.ChatModelAgentState{}

	newCtx, newState, err := middleware.AfterModelRewriteState(ctx, state, nil)

	assert.NoError(t, err)
	assert.Equal(t, ctx, newCtx)
	assert.Equal(t, state, newState)
}

// TestWrapInvokableToolCall tests that tool wrapping captures tool results
func TestWrapInvokableToolCall(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	t.Run("captures tool result in pending", func(t *testing.T) {
		runID := "test-run-123"
		ctx := context.WithValue(context.Background(), runIDKey, runID)

		// Set up pending for this run
		middleware.pendingSaves = map[string]*pendingConversation{
			runID: {
				inputMessages: []*schema.Message{{Role: schema.User, Content: "test"}},
				ctx:           context.Background(),
			},
		}

		endpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
			return `{"results": ["chunk1", "chunk2"]}`, nil
		}
		tCtx := &adk.ToolContext{Name: "semantic_search", CallID: "call-abc"}

		wrapped, err := middleware.WrapInvokableToolCall(ctx, endpoint, tCtx)

		assert.NoError(t, err)
		assert.NotNil(t, wrapped)

		result, err := wrapped(ctx, `{"query": "agentic rag"}`)
		assert.NoError(t, err)
		assert.Equal(t, `{"results": ["chunk1", "chunk2"]}`, result)

		// Verify tool result was captured in pending
		pending := middleware.pendingSaves[runID]
		assert.Len(t, pending.toolResults, 1)
		assert.Equal(t, schema.Tool, pending.toolResults[0].Role)
		assert.Equal(t, `{"results": ["chunk1", "chunk2"]}`, pending.toolResults[0].Content)
		assert.Equal(t, "call-abc", pending.toolResults[0].ToolCallID)
		assert.Equal(t, "semantic_search", pending.toolResults[0].ToolName)
	})

	t.Run("tool execution error does not capture result", func(t *testing.T) {
		runID := "test-run-err"
		ctx := context.WithValue(context.Background(), runIDKey, runID)

		middleware.pendingSaves = map[string]*pendingConversation{
			runID: {
				inputMessages: []*schema.Message{{Role: schema.User, Content: "test"}},
				ctx:           context.Background(),
			},
		}

		endpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
			return "", errors.New("tool failed")
		}
		tCtx := &adk.ToolContext{Name: "failing_tool", CallID: "call-err"}

		wrapped, err := middleware.WrapInvokableToolCall(ctx, endpoint, tCtx)
		assert.NoError(t, err)

		_, err = wrapped(ctx, "test")
		assert.Error(t, err)

		// No tool result should be captured on error
		pending := middleware.pendingSaves[runID]
		assert.Len(t, pending.toolResults, 0)
	})

	t.Run("no pending does not panic", func(t *testing.T) {
		ctx := context.Background()
		endpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
			return "result", nil
		}
		tCtx := &adk.ToolContext{Name: "some_tool", CallID: "call-noop"}

		wrapped, err := middleware.WrapInvokableToolCall(ctx, endpoint, tCtx)
		assert.NoError(t, err)

		// Should not panic even without pending
		result, err := wrapped(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, "result", result)
	})
}

// TestWrapStreamableToolCall tests that streaming tool wrapping is a no-op
func TestWrapStreamableToolCall(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	ctx := context.Background()
	endpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		return nil, nil
	}
	tCtx := &adk.ToolContext{}

	wrapped, err := middleware.WrapStreamableToolCall(ctx, endpoint, tCtx)

	assert.NoError(t, err)
	assert.NotNil(t, wrapped)
	// Verify it returns the same result as the original endpoint
	result, err := wrapped(ctx, "test")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestWrapEnhancedInvokableToolCall tests that enhanced tool wrapping is a no-op
func TestWrapEnhancedInvokableToolCall(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	ctx := context.Background()
	endpoint := func(ctx context.Context, toolArgument *schema.ToolArgument, opts ...tool.Option) (*schema.ToolResult, error) {
		return &schema.ToolResult{}, nil
	}
	tCtx := &adk.ToolContext{}

	wrapped, err := middleware.WrapEnhancedInvokableToolCall(ctx, endpoint, tCtx)

	assert.NoError(t, err)
	assert.NotNil(t, wrapped)
	// Verify it returns the same result as the original endpoint
	result, err := wrapped(ctx, &schema.ToolArgument{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestWrapEnhancedStreamableToolCall tests that enhanced streaming tool wrapping is a no-op
func TestWrapEnhancedStreamableToolCall(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	ctx := context.Background()
	endpoint := func(ctx context.Context, toolArgument *schema.ToolArgument, opts ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error) {
		return nil, nil
	}
	tCtx := &adk.ToolContext{}

	wrapped, err := middleware.WrapEnhancedStreamableToolCall(ctx, endpoint, tCtx)

	assert.NoError(t, err)
	assert.NotNil(t, wrapped)
	// Verify it returns the same result as the original endpoint
	result, err := wrapped(ctx, &schema.ToolArgument{})
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestWrapModel tests that WrapModel returns a conversationSavingModel wrapper
func TestWrapModel(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	ctx := context.Background()
	baseModel := &mockBaseChatModel{}
	mc := &adk.ModelContext{}

	wrapped, err := middleware.WrapModel(ctx, baseModel, mc)

	assert.NoError(t, err)
	assert.NotNil(t, wrapped)

	// Verify it's the correct wrapper type
	savingModel, ok := wrapped.(*conversationSavingModel)
	assert.True(t, ok, "Wrapped model should be *conversationSavingModel")
	assert.Equal(t, baseModel, savingModel.base)
	assert.Equal(t, middleware, savingModel.middleware)
	assert.Equal(t, mc, savingModel.modelContext)
}

// mockBaseChatModel is a test double for model.BaseChatModel
type mockBaseChatModel struct {
	generateFunc func(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error)
	streamFunc   func(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
}

func (m *mockBaseChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, messages, opts...)
	}
	return &schema.Message{Role: schema.Assistant, Content: "test response"}, nil
}

func (m *mockBaseChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, messages, opts...)
	}
	return nil, nil
}

// TestExtractFinishReason tests extracting finish reason from messages
func TestExtractFinishReason(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	tests := []struct {
		name         string
		message      *schema.Message
		expected     string
	}{
		{
			name:     "Nil message",
			message:  nil,
			expected: "",
		},
		{
			name: "Nil ResponseMeta",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
			},
			expected: "",
		},
		{
			name: "With finish reason",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "stop",
				},
			},
			expected: "stop",
		},
		{
			name: "With empty finish reason",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "",
				},
			},
			expected: "",
		},
		{
			name: "With length finish reason",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "length",
				},
			},
			expected: "length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.ExtractFinishReason(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractPromptTokens tests extracting prompt tokens from messages
func TestExtractPromptTokens(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	tests := []struct {
		name     string
		message  *schema.Message
		expected int
	}{
		{
			name:     "Nil message",
			message:  nil,
			expected: 0,
		},
		{
			name: "Nil ResponseMeta",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
			},
			expected: 0,
		},
		{
			name: "Nil Usage",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: nil,
				},
			},
			expected: 0,
		},
		{
			name: "With usage data",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: &schema.TokenUsage{
						PromptTokens:     100,
						CompletionTokens: 50,
						TotalTokens:      150,
					},
				},
			},
			expected: 100,
		},
		{
			name: "With zero prompt tokens",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: &schema.TokenUsage{
						PromptTokens:     0,
						CompletionTokens: 50,
						TotalTokens:      50,
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.ExtractPromptTokens(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractCompletionTokens tests extracting completion tokens from messages
func TestExtractCompletionTokens(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	tests := []struct {
		name     string
		message  *schema.Message
		expected int
	}{
		{
			name:     "Nil message",
			message:  nil,
			expected: 0,
		},
		{
			name: "Nil ResponseMeta",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
			},
			expected: 0,
		},
		{
			name: "Nil Usage",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: nil,
				},
			},
			expected: 0,
		},
		{
			name: "With usage data",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: &schema.TokenUsage{
						PromptTokens:     100,
						CompletionTokens: 50,
						TotalTokens:      150,
					},
				},
			},
			expected: 50,
		},
		{
			name: "With zero completion tokens",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: &schema.TokenUsage{
						PromptTokens:     100,
						CompletionTokens: 0,
						TotalTokens:      100,
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.ExtractCompletionTokens(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractTotalTokens tests extracting total tokens from messages
func TestExtractTotalTokens(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	tests := []struct {
		name     string
		message  *schema.Message
		expected int
	}{
		{
			name:     "Nil message",
			message:  nil,
			expected: 0,
		},
		{
			name: "Nil ResponseMeta",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
			},
			expected: 0,
		},
		{
			name: "Nil Usage",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: nil,
				},
			},
			expected: 0,
		},
		{
			name: "With usage data",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: &schema.TokenUsage{
						PromptTokens:     100,
						CompletionTokens: 50,
						TotalTokens:      150,
					},
				},
			},
			expected: 150,
		},
		{
			name: "With zero total tokens",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "test",
				ResponseMeta: &schema.ResponseMeta{
					Usage: &schema.TokenUsage{
						PromptTokens:     0,
						CompletionTokens: 0,
						TotalTokens:      0,
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.ExtractTotalTokens(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractResponseText tests extracting response text from messages
func TestExtractResponseText(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	tests := []struct {
		name     string
		message  *schema.Message
		expected string
	}{
		{
			name:     "Nil message",
			message:  nil,
			expected: "",
		},
		{
			name: "Simple text content",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "Hello, world!",
			},
			expected: "Hello, world!",
		},
		{
			name: "Empty content",
			message: &schema.Message{
				Role:    schema.Assistant,
				Content: "",
			},
			expected: "",
		},
		{
			name: "Reasoning content only",
			message: &schema.Message{
				Role:             schema.Assistant,
				Content:          "",
				ReasoningContent: "Let me think...",
			},
			expected: "Let me think...",
		},
		{
			name: "Multimodal content with text parts",
			message: &schema.Message{
				Role: schema.Assistant,
				AssistantGenMultiContent: []schema.MessageOutputPart{
					{
						Type: schema.ChatMessagePartTypeText,
						Text: "First part",
					},
					{
						Type: schema.ChatMessagePartTypeText,
						Text: "Second part",
					},
				},
			},
			expected: "First part\nSecond part",
		},
		{
			name: "Multimodal content with mixed parts",
			message: &schema.Message{
				Role: schema.Assistant,
				AssistantGenMultiContent: []schema.MessageOutputPart{
					{
						Type: schema.ChatMessagePartTypeText,
						Text: "Text content",
					},
					{
						Type: schema.ChatMessagePartTypeImageURL,
					},
				},
			},
			expected: "Text content",
		},
		{
			name: "Multimodal content with empty text parts",
			message: &schema.Message{
				Role: schema.Assistant,
				AssistantGenMultiContent: []schema.MessageOutputPart{
					{
						Type: schema.ChatMessagePartTypeText,
						Text: "",
					},
				},
			},
			expected: "",
		},
		{
			name: "Content takes precedence over reasoning",
			message: &schema.Message{
				Role:             schema.Assistant,
				Content:          "Direct content",
				ReasoningContent: "Hidden reasoning",
			},
			expected: "Direct content",
		},
		{
			name: "Empty multimodal content",
			message: &schema.Message{
				Role:                     schema.Assistant,
				AssistantGenMultiContent: []schema.MessageOutputPart{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.ExtractResponseText(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConversationSavingModel_Generate tests the Generate method
func TestConversationSavingModel_Generate(t *testing.T) {
	t.Run("Successful generation saves conversation", func(t *testing.T) {
		repo := &mockConversationRepository{
			saveFunc: func(ctx context.Context, conversation *entities.Conversation, messages []*entities.Message) error {
				// Verify conversation was built correctly
				assert.NotNil(t, conversation)
				assert.NotNil(t, messages)
				return nil
			},
		}
		log := logger.New(logger.LevelInfo, "text")
		middleware := &conversationMemoryMiddleware{
			conversationRepo: repo,
			logger:           log,
			saveTimeout:      10 * time.Second,
		}

		baseModel := &mockBaseChatModel{
			generateFunc: func(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
				return &schema.Message{
					Role:    schema.Assistant,
					Content: "Test response",
				}, nil
			},
		}

		wrapper := &conversationSavingModel{
			base:         baseModel,
			middleware:   middleware,
			modelContext: &adk.ModelContext{},
		}

		ctx := context.Background()
		messages := []*schema.Message{
			{Role: schema.User, Content: "Hello"},
		}

		resp, err := wrapper.Generate(ctx, messages)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "Test response", resp.Content)

		// Give async save time to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Generation error does not save", func(t *testing.T) {
		saveCalled := false
		repo := &mockConversationRepository{
			saveFunc: func(ctx context.Context, conversation *entities.Conversation, messages []*entities.Message) error {
				saveCalled = true
				return nil
			},
		}
		log := logger.New(logger.LevelInfo, "text")
		middleware := &conversationMemoryMiddleware{
			conversationRepo: repo,
			logger:           log,
			saveTimeout:      10 * time.Second,
		}

		baseModel := &mockBaseChatModel{
			generateFunc: func(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
				return nil, errors.New("generation failed")
			},
		}

		wrapper := &conversationSavingModel{
			base:         baseModel,
			middleware:   middleware,
			modelContext: &adk.ModelContext{},
		}

		ctx := context.Background()
		messages := []*schema.Message{
			{Role: schema.User, Content: "Hello"},
		}

		resp, err := wrapper.Generate(ctx, messages)

		assert.Error(t, err)
		assert.Nil(t, resp)

		// Give async save time (should not be called)
		time.Sleep(100 * time.Millisecond)
		assert.False(t, saveCalled, "Save should not be called on generation error")
	})
}

// TestBuildConversation tests building conversation entities
func TestBuildConversation(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := &conversationMemoryMiddleware{
		conversationRepo: repo,
		logger:           log,
		saveTimeout:      10 * time.Second,
	}

	t.Run("Build conversation with basic data", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")
		ctx = context.WithValue(ctx, "model", "gemini-pro")

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Hello"},
		}

		outputMessages := []*schema.Message{
			{
				Role:    schema.Assistant,
				Content: "Hi there!",
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "stop",
					Usage: &schema.TokenUsage{
						PromptTokens:     10,
						CompletionTokens: 5,
						TotalTokens:      15,
					},
				},
			},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.NotEmpty(t, conversation.ID)
		assert.Equal(t, "nb-123", *conversation.NotebookID)
		assert.Len(t, messages, 2) // One input + one output
		assert.Equal(t, "test-response-id", messages[0].ResponseID)
		assert.Equal(t, "stop", messages[1].FinishReason)
		assert.Equal(t, 10, messages[1].PromptTokens)
	})

	t.Run("Build conversation with previous response ID", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "previous_response_id", "prev-123")

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Continue"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "OK"},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 2)
	})

	t.Run("Build conversation without context values", func(t *testing.T) {
		ctx := context.Background()

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Test"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "Response"},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Nil(t, conversation.NotebookID)
		assert.Len(t, messages, 2)
	})

	t.Run("System role messages are filtered out", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")
		ctx = context.WithValue(ctx, "model", "gemini-pro")

		inputMessages := []*schema.Message{
			{Role: schema.System, Content: "You are a helpful assistant"},
			{Role: schema.User, Content: "Hello"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "Hi there!"},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 2) // Only user + assistant, system filtered out

		// Verify message roles
		roles := make([]string, len(messages))
		for i, msg := range messages {
			roles[i] = msg.Message.Role
		}
		assert.NotContains(t, roles, "system", "System messages should be filtered out")
		assert.Contains(t, roles, "user", "User messages should be preserved")
		assert.Contains(t, roles, "assistant", "Assistant messages should be preserved")
	})

	t.Run("System role messages filtered in output", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Test"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.System, Content: "System context"},
			{Role: schema.Assistant, Content: "Response"},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 2) // Only user + assistant, system filtered out

		// Verify message roles
		roles := make([]string, len(messages))
		for i, msg := range messages {
			roles[i] = msg.Message.Role
		}
		assert.NotContains(t, roles, "system", "System messages should be filtered out from output")
	})

	t.Run("Only system messages results in empty list", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")

		inputMessages := []*schema.Message{
			{Role: schema.System, Content: "System prompt"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.System, Content: "Another system message"},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 0, "All messages should be filtered out when only system messages are present")
	})

	t.Run("Existing conversation skips input messages", func(t *testing.T) {
		// Simulate an existing conversation by setting conversation_id in context
		// and having the mock repo return an existing conversation
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")
		ctx = context.WithValue(ctx, "conversation_id", "existing-conv-123")

		repoOrig := repo

		// We need to temporarily set up the mock for this test
		middlewareExisting := &conversationMemoryMiddleware{
			conversationRepo: &mockConversationRepository{
				findByIDFunc: func(ctx context.Context, id string) (*entities.Conversation, error) {
					return &entities.Conversation{ID: "existing-conv-123"}, nil
				},
				getMessagesFunc: func(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error) {
					// Return 2 existing messages → sequenceNum will be 3
					return []*entities.Message{
						{SequenceNum: 2, Message: &entities.StoredMessage{Role: "assistant"}},
						{SequenceNum: 1, Message: &entities.StoredMessage{Role: "user"}},
					}, nil
				},
			},
			logger:      logger.New(logger.LevelInfo, "text"),
			saveTimeout: 10 * time.Second,
		}

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Already saved user message"},
			{Role: schema.Assistant, Content: "Already saved response"},
			{Role: schema.User, Content: "This should NOT be saved"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "New response"},
		}

		conversation, messages, err := middlewareExisting.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 1, "Only output message should be saved for existing conversation")
		assert.Equal(t, "New response", messages[0].Message.Content)
		_ = repoOrig
	})

	t.Run("New conversation saves input and output", func(t *testing.T) {
		// No conversation_id or previous_response_id → new conversation (sequenceNum=1)
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Hello"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "Hi there!"},
		}

		conversation, messages, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 2, "Both input and output should be saved for new conversation")
		assert.Equal(t, "Hello", messages[0].Message.Content)
		assert.Equal(t, "Hi there!", messages[1].Message.Content)
	})

	t.Run("Multiple agent iterations only save output", func(t *testing.T) {
		// Simulate multiple agent iterations where each creates a new pending
		// The conversation already exists from iteration 1
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")
		ctx = context.WithValue(ctx, "conversation_id", "existing-conv-456")

		middlewareExisting := &conversationMemoryMiddleware{
			conversationRepo: &mockConversationRepository{
				findByIDFunc: func(ctx context.Context, id string) (*entities.Conversation, error) {
					return &entities.Conversation{ID: "existing-conv-456"}, nil
				},
				getMessagesFunc: func(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error) {
					return []*entities.Message{
						{SequenceNum: 2, Message: &entities.StoredMessage{Role: "assistant"}},
						{SequenceNum: 1, Message: &entities.StoredMessage{Role: "user"}},
					}, nil
				},
			},
			logger:      logger.New(logger.LevelInfo, "text"),
			saveTimeout: 10 * time.Second,
		}

		// Iteration 2: input includes user_msg + tool_calls (already saved), output is new
		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "what is agentic rag?"},
			{Role: schema.Assistant, Content: "tool call result"},
			{Role: schema.Tool, Content: "search result"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "Final answer about agentic RAG"},
		}

		conversation, messages, err := middlewareExisting.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Len(t, messages, 1, "Only output should be saved for subsequent iterations")
		assert.Equal(t, "Final answer about agentic RAG", messages[0].Message.Content)
		// Verify sequence number continues from existing
		assert.Equal(t, 3, messages[0].SequenceNum)
	})
}

// TestMergeStreamChunks tests that streaming chunks are merged properly including ToolCalls
func TestMergeStreamChunks(t *testing.T) {
	t.Run("empty chunks returns nil", func(t *testing.T) {
		result := mergeStreamChunks(nil)
		assert.Nil(t, result)
	})

	t.Run("single chunk returned as-is", func(t *testing.T) {
		chunk := &schema.Message{Role: schema.Assistant, Content: "hello"}
		result := mergeStreamChunks([]*schema.Message{chunk})
		assert.Equal(t, chunk, result)
	})

	t.Run("merges text content across chunks", func(t *testing.T) {
		chunks := []*schema.Message{
			{Role: schema.Assistant, Content: "Hello "},
			{Role: schema.Assistant, Content: "World"},
		}
		result := mergeStreamChunks(chunks)
		assert.Equal(t, "Hello World", result.Content)
		assert.Equal(t, schema.Assistant, result.Role)
	})

	t.Run("merges ToolCalls from streaming chunks", func(t *testing.T) {
		idx0, idx1 := 0, 1
		chunks := []*schema.Message{
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{Index: &idx0, ID: "call_1", Type: "function", Function: schema.FunctionCall{Name: "semantic_search", Arguments: `{"query": "ag`}},
				},
			},
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{Index: &idx0, ID: "call_1", Type: "function", Function: schema.FunctionCall{Name: "semantic_search", Arguments: `entic rag"}`}},
				},
			},
			{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{
					{Index: &idx1, ID: "call_2", Type: "function", Function: schema.FunctionCall{Name: "chunk_read", Arguments: `{"id": "abc"}`}},
				},
			},
		}
		result := mergeStreamChunks(chunks)
		assert.Equal(t, schema.Assistant, result.Role)
		require.Len(t, result.ToolCalls, 2)
		assert.Equal(t, "call_1", result.ToolCalls[0].ID)
		assert.Equal(t, "semantic_search", result.ToolCalls[0].Function.Name)
		assert.Equal(t, `{"query": "agentic rag"}`, result.ToolCalls[0].Function.Arguments)
		assert.Equal(t, "call_2", result.ToolCalls[1].ID)
		assert.Equal(t, "chunk_read", result.ToolCalls[1].Function.Name)
		assert.Equal(t, `{"id": "abc"}`, result.ToolCalls[1].Function.Arguments)
	})

	t.Run("merges ReasoningContent across chunks", func(t *testing.T) {
		chunks := []*schema.Message{
			{Role: schema.Assistant, ReasoningContent: "Let me think"},
			{Role: schema.Assistant, ReasoningContent: " about this..."},
		}
		result := mergeStreamChunks(chunks)
		assert.Equal(t, "Let me think about this...", result.ReasoningContent)
	})
}

// TestBuildConversationWithToolResults tests that buildConversation includes tool results
func TestBuildConversationWithToolResults(t *testing.T) {
	t.Run("tool results saved after output messages", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")

		mw := &conversationMemoryMiddleware{
			conversationRepo: &mockConversationRepository{},
			logger:           logger.New(logger.LevelInfo, "text"),
			saveTimeout:      10 * time.Second,
		}

		toolResults := []*schema.Message{
			schema.ToolMessage(`{"chunks": ["result1"]}`, "call-1", schema.WithToolName("semantic_search")),
			schema.ToolMessage("full text content", "call-2", schema.WithToolName("chunk_read")),
		}

		conversation, messages, err := mw.buildConversation(ctx, &pendingConversation{
			inputMessages: []*schema.Message{
				{Role: schema.User, Content: "what is agentic rag?"},
			},
			outputMessages: []*schema.Message{
				{
					Role:    schema.Assistant,
					Content: "Let me search for that.",
					ToolCalls: []schema.ToolCall{
						{ID: "call-1", Type: "function", Function: schema.FunctionCall{Name: "semantic_search"}},
						{ID: "call-2", Type: "function", Function: schema.FunctionCall{Name: "chunk_read"}},
					},
				},
			},
			toolResults:      toolResults,
			savedToolResults: 0,
			responseID:       "resp-123",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		require.Len(t, messages, 4) // user + assistant_with_TC + 2 tool results

		assert.Equal(t, "user", messages[0].Message.Role)
		assert.Equal(t, "assistant", messages[1].Message.Role)
		assert.Equal(t, "tool", messages[2].Message.Role)
		assert.Equal(t, "tool", messages[3].Message.Role)
	})

	t.Run("only unsaved tool results are saved", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")

		mw := &conversationMemoryMiddleware{
			conversationRepo: &mockConversationRepository{},
			logger:           logger.New(logger.LevelInfo, "text"),
			saveTimeout:      10 * time.Second,
		}

		toolResults := []*schema.Message{
			schema.ToolMessage("result1", "call-1", schema.WithToolName("search")),
			schema.ToolMessage("result2", "call-2", schema.WithToolName("read")),
			schema.ToolMessage("result3", "call-3", schema.WithToolName("search2")),
		}

		pending := &pendingConversation{
			inputMessages: []*schema.Message{
				{Role: schema.User, Content: "test"},
			},
			outputMessages: []*schema.Message{
				{Role: schema.Assistant, Content: "response"},
			},
			toolResults:      toolResults,
			savedToolResults: 1, // First tool result already saved
			responseID:       "resp-123",
		}

		conversation, messages, err := mw.buildConversation(ctx, pending)
		require.NoError(t, err)
		assert.NotNil(t, conversation)
		// user + assistant + 2 unsaved tool results (indices 1 and 2)
		require.Len(t, messages, 4)
		assert.Equal(t, "tool", messages[2].Message.Role)
		assert.Equal(t, "tool", messages[3].Message.Role)

		// Verify savedToolResults was updated
		assert.Equal(t, 3, pending.savedToolResults)
	})

	t.Run("existing conversation with tool results", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "notebook_id", "nb-123")
		ctx = context.WithValue(ctx, "conversation_id", "existing-conv")

		mw := &conversationMemoryMiddleware{
			conversationRepo: &mockConversationRepository{
				findByIDFunc: func(ctx context.Context, id string) (*entities.Conversation, error) {
					return &entities.Conversation{ID: "existing-conv"}, nil
				},
				getMessagesFunc: func(ctx context.Context, conversationID string, limit int, beforeSequence *int, isConversationHistory *bool) ([]*entities.Message, error) {
					return []*entities.Message{{SequenceNum: 2, Message: &entities.StoredMessage{Role: "assistant"}}}, nil
				},
			},
			logger:      logger.New(logger.LevelInfo, "text"),
			saveTimeout: 10 * time.Second,
		}

		toolResults := []*schema.Message{
			schema.ToolMessage("search results", "call-1", schema.WithToolName("semantic_search")),
		}

		conversation, messages, err := mw.buildConversation(ctx, &pendingConversation{
			inputMessages: []*schema.Message{
				{Role: schema.User, Content: "already saved"},
			},
			outputMessages: []*schema.Message{
				{Role: schema.Assistant, Content: "Let me search.", ToolCalls: []schema.ToolCall{{ID: "call-1"}}},
			},
			toolResults:      toolResults,
			savedToolResults: 0,
			responseID:       "resp-123",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		// Input skipped (existing conv) + 1 output + 1 tool result
		require.Len(t, messages, 2)
		assert.Equal(t, "assistant", messages[0].Message.Role)
		assert.Equal(t, "tool", messages[1].Message.Role)
		assert.Equal(t, 3, messages[0].SequenceNum)
		assert.Equal(t, 4, messages[1].SequenceNum)
	})
}

// TestExtractReasoningContent tests extracting reasoning content from message Extra
func TestExtractReasoningContent(t *testing.T) {
	t.Run("extracts from extra reasoning-content field", func(t *testing.T) {
		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: "search results",
			Extra: map[string]any{
				"reasoning-content": "I need to search for agentic RAG",
			},
		}
		extractReasoningContent(msg)
		assert.Equal(t, "I need to search for agentic RAG", msg.ReasoningContent)
	})

	t.Run("does not overwrite existing ReasoningContent", func(t *testing.T) {
		msg := &schema.Message{
			Role:             schema.Assistant,
			Content:          "content",
			ReasoningContent: "existing reasoning",
			Extra: map[string]any{
				"reasoning-content": "new reasoning",
			},
		}
		extractReasoningContent(msg)
		assert.Equal(t, "existing reasoning", msg.ReasoningContent)
	})

	t.Run("nil message does not panic", func(t *testing.T) {
		extractReasoningContent(nil)
	})

	t.Run("no extra field leaves ReasoningContent empty", func(t *testing.T) {
		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: "hello",
		}
		extractReasoningContent(msg)
		assert.Equal(t, "", msg.ReasoningContent)
	})

	t.Run("extra without reasoning key leaves ReasoningContent empty", func(t *testing.T) {
		msg := &schema.Message{
			Role: schema.Assistant,
			Extra: map[string]any{
				"other_key": "value",
			},
		}
		extractReasoningContent(msg)
		assert.Equal(t, "", msg.ReasoningContent)
	})

	t.Run("non-string extra value is ignored", func(t *testing.T) {
		msg := &schema.Message{
			Role: schema.Assistant,
			Extra: map[string]any{
				"reasoning-content": 12345,
			},
		}
		extractReasoningContent(msg)
		assert.Equal(t, "", msg.ReasoningContent)
	})

	t.Run("empty string extra value leaves ReasoningContent empty", func(t *testing.T) {
		msg := &schema.Message{
			Role: schema.Assistant,
			Extra: map[string]any{
				"reasoning-content": "",
			},
		}
		extractReasoningContent(msg)
		assert.Equal(t, "", msg.ReasoningContent)
	})
}