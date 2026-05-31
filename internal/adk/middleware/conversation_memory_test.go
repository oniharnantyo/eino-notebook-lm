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
	saveFunc             func(ctx context.Context, conversation *entities.Conversation) error
}

func (m *mockConversationRepository) Save(ctx context.Context, conversation *entities.Conversation) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, conversation)
	}
	return nil
}

func (m *mockConversationRepository) FindByResponseID(ctx context.Context, responseID string) (*entities.Conversation, error) {
	if m.findByResponseIDFunc != nil {
		return m.findByResponseIDFunc(ctx, responseID)
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

// TestNewConversationMemory tests the constructor
func TestNewConversationMemory(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")

	middleware := NewConversationMemory(repo, log)

	assert.NotNil(t, middleware)
	assert.Equal(t, repo, middleware.conversationRepo)
	assert.Equal(t, log, middleware.logger)
	assert.Equal(t, 10*time.Second, middleware.saveTimeout)
}

// TestSetSaveTimeout tests setting the save timeout
func TestSetSaveTimeout(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

	// Test setting various timeouts
	testTimeouts := []time.Duration{
		5 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		0,
	}

	for _, timeout := range testTimeouts {
		middleware.SetSaveTimeout(timeout)
		assert.Equal(t, timeout, middleware.saveTimeout)
	}
}

// TestBeforeAgent tests that BeforeAgent is a no-op
func TestBeforeAgent(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

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
	middleware := NewConversationMemory(repo, log)

	tests := []struct {
		name              string
		previousResponseID interface{}
		mockConversation  *entities.Conversation
		mockError         error
		expectMessages    int
		expectError       bool
	}{
		{
			name:              "No previous response ID",
			previousResponseID: "",
			mockConversation:  nil,
			mockError:         nil,
			expectMessages:    0,
			expectError:       false,
		},
		{
			name:              "Previous response ID not a string",
			previousResponseID: 123,
			mockConversation:  nil,
			mockError:         nil,
			expectMessages:    0,
			expectError:       false,
		},
		{
			name:              "Load conversation successfully",
			previousResponseID: "resp-123",
			mockConversation: &entities.Conversation{
				ResponseID: "resp-123",
				Messages: []*entities.StoredMessage{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there"},
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
			mockError:         nil,
			expectMessages:    0,
			expectError:       false,
		},
		{
			name:              "Load conversation fails",
			previousResponseID: "resp-error",
			mockConversation:  nil,
			mockError:         errors.New("database error"),
			expectMessages:    0,
			expectError:       false, // Should gracefully degrade
		},
		{
			name:              "Load conversation with empty messages",
			previousResponseID: "resp-empty",
			mockConversation: &entities.Conversation{
				ResponseID: "resp-empty",
				Messages:   []*entities.StoredMessage{},
			},
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

			assert.Equal(t, ctx, newCtx)
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
	middleware := NewConversationMemory(repo, log)

	ctx := context.Background()
	state := &adk.ChatModelAgentState{}

	newCtx, newState, err := middleware.AfterModelRewriteState(ctx, state, nil)

	assert.NoError(t, err)
	assert.Equal(t, ctx, newCtx)
	assert.Equal(t, state, newState)
}

// TestWrapInvokableToolCall tests that tool wrapping is a no-op
func TestWrapInvokableToolCall(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

	ctx := context.Background()
	endpoint := func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		return "result", nil
	}
	tCtx := &adk.ToolContext{}

	wrapped, err := middleware.WrapInvokableToolCall(ctx, endpoint, tCtx)

	assert.NoError(t, err)
	assert.NotNil(t, wrapped)
	// Verify it returns the same result as the original endpoint
	result, err := wrapped(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, "result", result)
}

// TestWrapStreamableToolCall tests that streaming tool wrapping is a no-op
func TestWrapStreamableToolCall(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

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
	middleware := NewConversationMemory(repo, log)

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
	middleware := NewConversationMemory(repo, log)

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
	middleware := NewConversationMemory(repo, log)

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
	middleware := NewConversationMemory(repo, log)

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
			result := middleware.extractFinishReason(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractPromptTokens tests extracting prompt tokens from messages
func TestExtractPromptTokens(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

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
			result := middleware.extractPromptTokens(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractCompletionTokens tests extracting completion tokens from messages
func TestExtractCompletionTokens(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

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
			result := middleware.extractCompletionTokens(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractTotalTokens tests extracting total tokens from messages
func TestExtractTotalTokens(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

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
			result := middleware.extractTotalTokens(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractResponseText tests extracting response text from messages
func TestExtractResponseText(t *testing.T) {
	repo := &mockConversationRepository{}
	log := logger.New(logger.LevelInfo, "text")
	middleware := NewConversationMemory(repo, log)

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
			result := middleware.extractResponseText(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConversationSavingModel_Generate tests the Generate method
func TestConversationSavingModel_Generate(t *testing.T) {
	t.Run("Successful generation saves conversation", func(t *testing.T) {
		repo := &mockConversationRepository{
			saveFunc: func(ctx context.Context, conversation *entities.Conversation) error {
				// Verify conversation was built correctly
				assert.NotNil(t, conversation)
				assert.NotEmpty(t, conversation.ResponseID)
				assert.NotEmpty(t, conversation.Messages)
				return nil
			},
		}
		log := logger.New(logger.LevelInfo, "text")
		middleware := NewConversationMemory(repo, log)

		baseModel := &mockBaseChatModel{
			generateFunc: func(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
				return &schema.Message{
					Role:    schema.Assistant,
					Content: "Test response",
				}, nil
			},
		}

		wrapper := &conversationSavingModel{
			base:        baseModel,
			middleware:  middleware,
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
			saveFunc: func(ctx context.Context, conversation *entities.Conversation) error {
				saveCalled = true
				return nil
			},
		}
		log := logger.New(logger.LevelInfo, "text")
		middleware := NewConversationMemory(repo, log)

		baseModel := &mockBaseChatModel{
			generateFunc: func(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
				return nil, errors.New("generation failed")
			},
		}

		wrapper := &conversationSavingModel{
			base:        baseModel,
			middleware:  middleware,
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
	middleware := NewConversationMemory(repo, log)

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

		conversation, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.NotEmpty(t, conversation.ID)
		assert.NotEmpty(t, conversation.ResponseID)
		assert.Equal(t, "nb-123", *conversation.NotebookID)
		assert.Nil(t, conversation.PreviousResponseID)
		assert.Equal(t, "gemini-pro", conversation.Model)
		assert.Equal(t, "Hi there!", conversation.ResponseText)
		assert.Equal(t, "stop", conversation.FinishReason)
		assert.Equal(t, 10, conversation.PromptTokens)
		assert.Equal(t, 5, conversation.CompletionTokens)
		assert.Equal(t, 15, conversation.TotalTokens)
		assert.Len(t, conversation.Messages, 2) // One input + one output
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

		conversation, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.NotNil(t, conversation.PreviousResponseID)
		assert.Equal(t, "prev-123", *conversation.PreviousResponseID)
	})

	t.Run("Build conversation without context values", func(t *testing.T) {
		ctx := context.Background()

		inputMessages := []*schema.Message{
			{Role: schema.User, Content: "Test"},
		}

		outputMessages := []*schema.Message{
			{Role: schema.Assistant, Content: "Response"},
		}

		conversation, err := middleware.buildConversation(ctx, &pendingConversation{
			inputMessages:  inputMessages,
			outputMessages: outputMessages,
			responseID:     "test-response-id",
		})

		require.NoError(t, err)
		assert.NotNil(t, conversation)
		assert.Nil(t, conversation.NotebookID)
		assert.Nil(t, conversation.PreviousResponseID)
		assert.Equal(t, "unknown", conversation.Model)
	})
}
