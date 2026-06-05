package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/chat"
	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/sse"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// ChatCompletionsHandler handles OpenAI-compatible chat completion HTTP requests
type ChatCompletionsHandler struct {
	useCase   chat.ResponseUseCase
	logger    *logger.Logger
	validator *validator.Validate
}

// NewChatCompletionsHandler creates a new chat completions handler
func NewChatCompletionsHandler(
	useCase chat.ResponseUseCase,
	log *logger.Logger,
) *ChatCompletionsHandler {
	return &ChatCompletionsHandler{
		useCase:   useCase,
		logger:    log,
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
}

// CreateCompletion handles chat completion requests
// POST /api/v1/chat/completions
// This method provides OpenAI-compatible chat completion endpoint
func (h *ChatCompletionsHandler) CreateCompletion(w http.ResponseWriter, r *http.Request) {
	var req dtos.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("failed to decode request: %s", err.Error()))
		return
	}

	// Validate request using validator
	if err := h.validator.Struct(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("validation failed: %s", err.Error()))
		return
	}

	// Validate messages array is not empty
	if len(req.Messages) == 0 {
		h.respondWithError(w, http.StatusBadRequest, "invalid_request", "messages array cannot be empty")
		return
	}

	// Convert ChatCompletionRequest to ResponseRequest
	responseReq := h.convertToResponseRequest(&req)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Check if client supports streaming
	if _, ok := w.(http.Flusher); !ok {
		h.respondWithError(w, http.StatusNotImplemented, "internal_error", "streaming not supported")
		return
	}

	stream, meta, err := h.useCase.Stream(r.Context(), responseReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	formatter := sse.NewChatCompletionsFormatter()
	if err := formatter.WriteResponse(w, stream, meta); err != nil {
		h.logger.Error("Failed to write SSE response", "error", err)
	}
}

// convertToResponseRequest converts a ChatCompletionRequest to a ResponseRequest
func (h *ChatCompletionsHandler) convertToResponseRequest(req *dtos.ChatCompletionRequest) *dtos.ResponseRequest {
	// Convert messages to input items
	var input []dtos.ItemParam
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			input = append(input, &dtos.UserMessageItemParam{
				Type:    "message",
				Role:    "user",
				Content: msg.Content,
			})
		case "assistant":
			input = append(input, &dtos.AssistantMessageItemParam{
				Type:    "message",
				Role:    "assistant",
				Content: msg.Content,
			})
		case "system":
			input = append(input, &dtos.SystemMessageItemParam{
				Type:    "message",
				Role:    "system",
				Content: msg.Content,
			})
		}
	}

	// Convert tools
	var tools []dtos.ResponsesTool
	for _, tool := range req.Tools {
		tools = append(tools, dtos.ResponsesTool{
			Type:        tool.Type,
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
			Strict:      tool.Function.Strict,
		})
	}

	// Map max_tokens/max_completion_tokens to max_output_tokens
	var maxOutputTokens *int
	if req.MaxCompletionTokens != nil {
		maxOutputTokens = req.MaxCompletionTokens
	} else if req.MaxTokens != nil {
		maxOutputTokens = req.MaxTokens
	}

	// Build SourceIDs array from extra_body source_id
	var sourceIDs []string
	if req.SourceID != nil {
		sourceIDs = []string{*req.SourceID}
	}

	return &dtos.ResponseRequest{
		Input:           input,
		Model:           req.Model,
		Temperature:     req.Temperature,
		MaxOutputTokens: maxOutputTokens,
		Stream:          req.Stream,
		Tools:           tools,
		ToolChoice:      req.ToolChoice,
		NotebookID:      req.NotebookID,
		ConversationID:  req.ConversationID,
		SourceIDs:       sourceIDs, // Pass source_id to RAG pipeline
	}
}


// respondWithError writes an error response in OpenAI format
func (h *ChatCompletionsHandler) respondWithError(w http.ResponseWriter, code int, errType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"message": message,
			"type":    errType,
			"code":    getErrorCode(code),
		},
	}); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}
