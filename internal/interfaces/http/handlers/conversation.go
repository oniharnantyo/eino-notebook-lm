package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	conversationUseCase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/conversation"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// ConversationHandler handles conversation HTTP requests
type ConversationHandler struct {
	useCase conversationUseCase.ConversationUseCase
	logger  *logger.Logger
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(
	useCase conversationUseCase.ConversationUseCase,
	log *logger.Logger,
) *ConversationHandler {
	return &ConversationHandler{
		useCase: useCase,
		logger:  log,
	}
}

// ListByNotebook handles list conversations for a specific notebook
// GET /api/v1/notebooks/{notebookId}/conversations
func (h *ConversationHandler) ListByNotebook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notebookID := vars["notebookId"]

	// Parse query parameters
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	model := query.Get("model")

	req := &dtos.ListConversationsRequest{
		Page:       page,
		Limit:      limit,
		NotebookID: notebookID,
		Model:      model,
	}

	result, err := h.useCase.List(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list conversations: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// GetMessages handles fetching paginated messages for a conversation
// GET /api/v1/notebooks/{notebookId}/conversations/{conversationId}/messages
func (h *ConversationHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notebookID := vars["notebookId"]
	conversationID := vars["conversationId"]

	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))

	var beforeSequence *int
	if seqStr := query.Get("before_sequence"); seqStr != "" {
		if seq, err := strconv.Atoi(seqStr); err == nil {
			beforeSequence = &seq
		}
	}

	req := &dtos.GetMessagesRequest{
		NotebookID:     notebookID,
		ConversationID: conversationID,
		Limit:          limit,
		BeforeSequence: beforeSequence,
	}

	result, err := h.useCase.GetMessages(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get messages: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// respondWithJSON writes a JSON response
func (h *ConversationHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError writes an error response
func (h *ConversationHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, dtos.ErrorResponse{
		Code:    http.StatusText(code),
		Message: message,
	})
}
