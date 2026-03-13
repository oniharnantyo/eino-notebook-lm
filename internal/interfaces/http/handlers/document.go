package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// DocumentHandler handles document HTTP requests
type DocumentHandler struct {
	useCase usecases.DocumentUseCase
	logger  *logger.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(useCase usecases.DocumentUseCase, log *logger.Logger) *DocumentHandler {
	return &DocumentHandler{
		useCase: useCase,
		logger:  log,
	}
}

// Create handles document creation requests
// TODO: Implement handler logic
func (h *DocumentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dtos.CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Call use case and handle response
	h.respondWithJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// GetByID handles get document by ID requests
// TODO: Implement handler logic
func (h *DocumentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Call use case and handle response
	h.logger.Info("get document by id", "id", id)
	h.respondWithJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// List handles list documents requests
// TODO: Implement handler logic
func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: Call use case and handle response
	h.respondWithJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// Update handles document update requests
// TODO: Implement handler logic
func (h *DocumentHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req dtos.UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// TODO: Call use case and handle response
	h.logger.Info("update document", "id", id)
	h.respondWithJSON(w, http.StatusNotImplemented, map[string]string{"message": "Not implemented"})
}

// Delete handles document deletion requests
// TODO: Implement handler logic
func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Call use case and handle response
	h.logger.Info("delete document", "id", id)
	w.WriteHeader(http.StatusNotImplemented)
}

// respondWithJSON writes a JSON response
func (h *DocumentHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError writes an error response
func (h *DocumentHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, dtos.ErrorResponse{
		Code:    http.StatusText(code),
		Message: message,
	})
}
