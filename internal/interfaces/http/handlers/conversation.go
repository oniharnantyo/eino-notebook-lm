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
	previousResponseID := query.Get("previous_response_id")

	req := &dtos.ListConversationsRequest{
		Page:               page,
		Limit:              limit,
		NotebookID:         notebookID,
		Model:              model,
		PreviousResponseID: previousResponseID,
	}

	result, err := h.useCase.List(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list conversations: %v", err))
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
