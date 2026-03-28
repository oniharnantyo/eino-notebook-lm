package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/notebook"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// NotebookHandler handles notebook HTTP requests
type NotebookHandler struct {
	useCase notebook.NotebookUseCase
	logger  *logger.Logger
}

// NewNotebookHandler creates a new notebook handler
func NewNotebookHandler(useCase notebook.NotebookUseCase, log *logger.Logger) *NotebookHandler {
	return &NotebookHandler{
		useCase: useCase,
		logger:  log,
	}
}

// Create handles notebook creation requests
func (h *NotebookHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dtos.CreateNotebookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	notebook, err := h.useCase.Create(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create notebook", "error", err)
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusCreated, notebook)
}

// GetByID handles get notebook by ID requests
func (h *NotebookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	notebook, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get notebook", "id", id, "error", err)
		h.respondWithError(w, http.StatusNotFound, "Notebook not found")
		return
	}

	h.respondWithJSON(w, http.StatusOK, notebook)
}

// List handles list notebooks requests
func (h *NotebookHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	status := query.Get("status")
	searchQuery := query.Get("q")

	req := &dtos.ListNotebooksRequest{
		Page:   page,
		Limit:  limit,
		Status: status,
		Query:  searchQuery,
	}

	result, err := h.useCase.List(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to list notebooks", "error", err)
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// Update handles notebook update requests
func (h *NotebookHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req dtos.UpdateNotebookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse ID from URL
	uid, err := uuid.Parse(id)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid notebook ID")
		return
	}
	req.ID = uid

	notebook, err := h.useCase.Update(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to update notebook", "id", id, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondWithJSON(w, http.StatusOK, notebook)
}

// Delete handles notebook deletion requests
func (h *NotebookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.useCase.Delete(r.Context(), id); err != nil {
		h.logger.Error("failed to delete notebook", "id", id, "error", err)
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// respondWithJSON writes a JSON response
func (h *NotebookHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError writes an error response
func (h *NotebookHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, dtos.ErrorResponse{
		Code:    http.StatusText(code),
		Message: message,
	})
}
