package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/sse"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper functions (avoiding collision with conversation_test.go's intPtr)
func strPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

type mockChatUseCase struct {
	mock.Mock
}

func (m *mockChatUseCase) Stream(ctx context.Context, req *dtos.ResponseRequest) (*schema.StreamReader[*schema.Message], *sse.StreamMeta, error) {
	args := m.Called(ctx, req)
	var sr *schema.StreamReader[*schema.Message]
	if args.Get(0) != nil {
		sr = args.Get(0).(*schema.StreamReader[*schema.Message])
	}
	var meta *sse.StreamMeta
	if args.Get(1) != nil {
		meta = args.Get(1).(*sse.StreamMeta)
	}
	return sr, meta, args.Error(2)
}

func TestChatCompletionsHandler_CreateCompletion_Success(t *testing.T) {
	// Arrange
	mockUC := new(mockChatUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(mockUC, log)

	// Create valid chat completion request with proper UUIDs
	notebookID := "123e4567-e89b-12d3-a456-426614174000"
	conversationID := "123e4567-e89b-12d3-a456-426614174001"
	sourceID := "123e4567-e89b-12d3-a456-426614174002"

	reqBody := dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
			{Role: "system", Content: "You are helpful"},
		},
		Model:               strPtr("gpt-4"),
		Temperature:         float64Ptr(0.7),
		MaxCompletionTokens: intPtr(1000),
		Stream:              true,
		Tools: []dtos.ChatCompletionTool{
			{
				Type: "function",
				Function: dtos.ChatCompletionFunction{
					Name:        "search",
					Description: strPtr("Search the web"),
					Parameters:  map[string]interface{}{"type": "object"},
				},
			},
		},
		NotebookID:     &notebookID,
		ConversationID: &conversationID,
		SourceID:       &sourceID,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Prepare mock stream
	pr, pw := schema.Pipe[*schema.Message](10)
	go func() {
		_ = pw.Send(&schema.Message{Content: "Response"}, nil)
		pw.Close()
	}()

	meta := &sse.StreamMeta{
		ResponseID: "resp_1",
	}

	mockUC.On("Stream", mock.Anything, mock.MatchedBy(func(r *dtos.ResponseRequest) bool {
		// Verify conversion was correct
		assert.NotNil(t, r.Input)
		assert.Equal(t, "gpt-4", *r.Model)
		assert.Equal(t, 0.7, *r.Temperature)
		assert.Equal(t, 1000, *r.MaxOutputTokens)
		assert.True(t, r.Stream)
		assert.Len(t, r.Tools, 1)
		assert.Equal(t, "search", r.Tools[0].Name)
		assert.Equal(t, notebookID, *r.NotebookID)
		assert.Equal(t, conversationID, *r.ConversationID)
		assert.NotNil(t, r.SourceIDs)
		assert.Len(t, r.SourceIDs, 1)
		assert.Equal(t, sourceID, r.SourceIDs[0])
		return true
	})).Return(pr, meta, nil)

	// Act
	handler.CreateCompletion(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	assert.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))

	mockUC.AssertExpectations(t)
}

func TestChatCompletionsHandler_CreateCompletion_InvalidJSON(t *testing.T) {
	// Arrange
	mockUC := new(mockChatUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(mockUC, log)

	// Invalid JSON
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	// Act
	handler.CreateCompletion(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errResp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.NotNil(t, errResp["error"])

	errObj := errResp["error"].(map[string]any)
	assert.Equal(t, "invalid_request", errObj["type"])
	assert.Contains(t, errObj["message"].(string), "failed to decode request")
}

func TestChatCompletionsHandler_CreateCompletion_EmptyMessages(t *testing.T) {
	// Arrange
	mockUC := new(mockChatUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(mockUC, log)

	// Empty messages array - validator catches this first
	reqBody := dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Act
	handler.CreateCompletion(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errResp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.NotNil(t, errResp["error"])

	errObj := errResp["error"].(map[string]any)
	assert.Equal(t, "invalid_request", errObj["type"])
	// The validator catches empty arrays with "required,min=1" tag
	assert.Contains(t, errObj["message"].(string), "validation failed")
}

func TestChatCompletionsHandler_CreateCompletion_ValidationError(t *testing.T) {
	// Arrange
	mockUC := new(mockChatUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(mockUC, log)

	// Invalid temperature value (out of range)
	temp := 3.0 // max is 2
	reqBody := dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: &temp,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Act
	handler.CreateCompletion(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errResp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.NotNil(t, errResp["error"])

	errObj := errResp["error"].(map[string]any)
	assert.Equal(t, "invalid_request", errObj["type"])
	assert.Contains(t, errObj["message"].(string), "validation failed")
}

func TestChatCompletionsHandler_CreateCompletion_UseCaseError(t *testing.T) {
	// Arrange
	mockUC := new(mockChatUseCase)
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(mockUC, log)

	reqBody := dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/chat/completions", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	mockUC.On("Stream", mock.Anything, mock.Anything).Return(nil, (*sse.StreamMeta)(nil), assert.AnError)

	// Act
	handler.CreateCompletion(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var errResp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.NotNil(t, errResp["error"])

	errObj := errResp["error"].(map[string]any)
	assert.Equal(t, "internal_error", errObj["type"])

	mockUC.AssertExpectations(t)
}

func TestChatCompletionsHandler_convertToResponseRequest_AllFields(t *testing.T) {
	// Arrange
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(nil, log)

	notebookID := "123e4567-e89b-12d3-a456-426614174000"
	conversationID := "123e4567-e89b-12d3-a456-426614174001"
	sourceID := "123e4567-e89b-12d3-a456-426614174002"

	req := &dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
			{Role: "system", Content: "You are helpful"},
		},
		Model:               strPtr("gpt-4"),
		Temperature:         float64Ptr(0.7),
		MaxCompletionTokens: intPtr(1000),
		Stream:              true,
		Tools: []dtos.ChatCompletionTool{
			{
				Type: "function",
				Function: dtos.ChatCompletionFunction{
					Name:        "search",
					Description: strPtr("Search the web"),
					Parameters:  map[string]interface{}{"type": "object"},
					Strict:      boolPtr(true),
				},
			},
		},
		NotebookID:     &notebookID,
		ConversationID: &conversationID,
		SourceID:       &sourceID,
	}

	// Act
	result := handler.convertToResponseRequest(req)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, "gpt-4", *result.Model)
	assert.Equal(t, 0.7, *result.Temperature)
	assert.Equal(t, 1000, *result.MaxOutputTokens)
	assert.True(t, result.Stream)
	assert.Equal(t, notebookID, *result.NotebookID)
	assert.Equal(t, conversationID, *result.ConversationID)
	assert.NotNil(t, result.SourceIDs)
	assert.Len(t, result.SourceIDs, 1)
	assert.Equal(t, sourceID, result.SourceIDs[0])

	// Verify messages conversion - Input is interface{} but should be []ItemParam
	inputItems, ok := result.Input.([]dtos.ItemParam)
	assert.True(t, ok)
	assert.Len(t, inputItems, 3)

	// First message (user)
	userMsg, ok := inputItems[0].(*dtos.UserMessageItemParam)
	assert.True(t, ok)
	assert.Equal(t, "user", userMsg.Role)
	assert.Equal(t, "message", userMsg.Type)
	assert.Equal(t, "Hello", userMsg.Content)

	// Second message (assistant)
	assistMsg, ok := inputItems[1].(*dtos.AssistantMessageItemParam)
	assert.True(t, ok)
	assert.Equal(t, "assistant", assistMsg.Role)
	assert.Equal(t, "message", assistMsg.Type)
	assert.Equal(t, "Hi there", assistMsg.Content)

	// Third message (system)
	sysMsg, ok := inputItems[2].(*dtos.SystemMessageItemParam)
	assert.True(t, ok)
	assert.Equal(t, "system", sysMsg.Role)
	assert.Equal(t, "message", sysMsg.Type)
	assert.Equal(t, "You are helpful", sysMsg.Content)

	// Verify tools conversion
	assert.Len(t, result.Tools, 1)
	assert.Equal(t, "function", result.Tools[0].Type)
	assert.Equal(t, "search", result.Tools[0].Name)
	assert.Equal(t, "Search the web", *result.Tools[0].Description)
	assert.NotNil(t, result.Tools[0].Parameters)
	assert.True(t, *result.Tools[0].Strict)
}

func TestChatCompletionsHandler_convertToResponseRequest_MaxTokensFallback(t *testing.T) {
	// Arrange
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(nil, log)

	req := &dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: intPtr(500),
	}

	// Act
	result := handler.convertToResponseRequest(req)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, 500, *result.MaxOutputTokens)
}

func TestChatCompletionsHandler_convertToResponseRequest_NoOptionalFields(t *testing.T) {
	// Arrange
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(nil, log)

	req := &dtos.ChatCompletionRequest{
		Messages: []dtos.ChatCompletionMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Act
	result := handler.convertToResponseRequest(req)

	// Assert
	assert.NotNil(t, result)
	assert.Nil(t, result.Model)
	assert.Nil(t, result.Temperature)
	assert.Nil(t, result.MaxOutputTokens)
	assert.False(t, result.Stream)
	assert.Nil(t, result.NotebookID)
	assert.Nil(t, result.ConversationID)
	assert.Empty(t, result.Metadata)
	assert.Empty(t, result.Tools)
}


func TestChatCompletionsHandler_respondWithError_OpenAIFormat(t *testing.T) {
	// Arrange
	log := logger.NewWithWriter(io.Discard, logger.LevelInfo, "text")
	handler := NewChatCompletionsHandler(nil, log)

	w := httptest.NewRecorder()

	// Act
	handler.respondWithError(w, http.StatusBadRequest, "invalid_request", "Test error message")

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	assert.NotNil(t, resp["error"])
	errObj := resp["error"].(map[string]any)
	assert.Equal(t, "Test error message", errObj["message"])
	assert.Equal(t, "invalid_request", errObj["type"])
	assert.NotNil(t, errObj["code"])
}
