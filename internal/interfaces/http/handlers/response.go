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

type ResponseHandler struct {
	useCase   chat.ResponseUseCase
	logger    *logger.Logger
	validator *validator.Validate
}

func NewResponseHandler(useCase chat.ResponseUseCase, log *logger.Logger) *ResponseHandler {
	return &ResponseHandler{
		useCase:   useCase,
		logger:    log,
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *ResponseHandler) CreateResponse(w http.ResponseWriter, r *http.Request) {
	var req dtos.ResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("failed to decode request: %s", err.Error()))
		return
	}

	// Validate request using govalidator
	if err := h.validator.Struct(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("validation failed: %s", err.Error()))
		return
	}

	// Validate input is not empty string
	if inputStr, ok := req.Input.(string); ok && inputStr == "" {
		h.respondWithError(w, http.StatusBadRequest, "invalid_request", "input cannot be empty")
		return
	}

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

	stream, meta, err := h.useCase.Stream(r.Context(), &req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	formatter := sse.NewResponsesAPIFormatter()
	if err := formatter.WriteResponse(w, stream, meta); err != nil {
		h.logger.Error("Failed to write SSE response", "error", err)
	}
}

func (h *ResponseHandler) respondWithError(w http.ResponseWriter, code int, errType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"type":    errType,
			"message": message,
			"code":    getErrorCode(code),
		},
	}); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}

func getErrorCode(code int) string {
	switch code {
	case http.StatusBadRequest:
		return "invalid_request"
	case http.StatusUnauthorized:
		return "invalid_api_key"
	case http.StatusNotFound:
		return "resource_not_found"
	default:
		return "internal_error"
	}
}
