package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
	sourceUseCase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/source"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// SourceHandler handles source HTTP requests
type SourceHandler struct {
	useCase sourceUseCase.SourceUseCase
	logger  *logger.Logger
}

// NewSourceHandler creates a new source handler
func NewSourceHandler(
	useCase sourceUseCase.SourceUseCase,
	log *logger.Logger,
) *SourceHandler {
	return &SourceHandler{
		useCase: useCase,
		logger:  log,
	}
}

// GetByID handles get source by ID requests
// GET /api/v1/notebooks/{notebookId}/sources/{id}
func (h *SourceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	source, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, fmt.Sprintf("Source not found: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, source)
}

// List handles list sources requests
// GET /api/v1/notebooks/{notebookId}/sources
func (h *SourceHandler) List(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notebookIDStr := vars["notebookId"]

	notebookID, err := mappers.ParseID(notebookIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid notebook_id format")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	contentType := query.Get("content_type")

	req := &dtos.ListSourcesRequest{
		NotebookID:  notebookID,
		Page:        page,
		Limit:       limit,
		ContentType: contentType,
	}

	result, err := h.useCase.List(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list sources: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// Delete handles source deletion requests
// DELETE /api/v1/notebooks/{notebookId}/sources/{id}
func (h *SourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.useCase.Delete(r.Context(), id); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete source: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondWithJSON writes a JSON response
func (h *SourceHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError writes an error response
func (h *SourceHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, dtos.ErrorResponse{
		Code:    http.StatusText(code),
		Message: message,
	})
}