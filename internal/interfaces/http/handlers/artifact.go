package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
	artifactUseCase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/artifact"
	mindmapUseCase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/mindmap"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ArtifactHandler handles artifact HTTP requests
type ArtifactHandler struct {
	artifactUseCase artifactUseCase.ArtifactUseCase
	mindmapUseCase  mindmapUseCase.MindmapUseCase
	logger          *logger.Logger
}

// NewArtifactHandler creates a new artifact handler
func NewArtifactHandler(
	artifactUC artifactUseCase.ArtifactUseCase,
	mindmapUC mindmapUseCase.MindmapUseCase,
	log *logger.Logger,
) *ArtifactHandler {
	return &ArtifactHandler{
		artifactUseCase: artifactUC,
		mindmapUseCase:  mindmapUC,
		logger:          log,
	}
}

// GenerateMindmap handles mindmap generation requests
// POST /api/v1/notebooks/{notebookId}/mindmap
func (h *ArtifactHandler) GenerateMindmap(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notebookIDStr := vars["notebookId"]

	notebookID, err := mappers.ParseID(notebookIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid notebook_id format")
		return
	}

	// Parse request body
	var req struct {
		SourceIDs []string `json:"source_ids"`
		Title     string   `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate source_ids
	if len(req.SourceIDs) == 0 {
		h.respondWithError(w, http.StatusBadRequest, "source_ids is required")
		return
	}

	// Parse source IDs
	sourceIDs := make([]uuid.UUID, 0, len(req.SourceIDs))
	for _, sourceIDStr := range req.SourceIDs {
		sourceID, err := mappers.ParseID(sourceIDStr)
		if err != nil {
			h.respondWithError(w, http.StatusBadRequest, "invalid source_id format")
			return
		}
		sourceIDs = append(sourceIDs, sourceID)
	}

	// Validate title
	title := req.Title
	if title == "" {
		title = "Mindmap"
	}

	mindmapReq := &dtos.TriggerMindmapRequest{
		NotebookID: notebookID,
		SourceIDs:  sourceIDs,
		Title:      title,
	}

	result, err := h.mindmapUseCase.Generate(r.Context(), mindmapReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate mindmap: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusAccepted, result)
}

// GetByID handles get artifact by ID requests
// GET /api/v1/notebooks/{notebookId}/artifacts/{id}
func (h *ArtifactHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	artifact, err := h.artifactUseCase.GetByID(r.Context(), id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, fmt.Sprintf("Artifact not found: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, artifact)
}

// List handles list artifacts requests
// GET /api/v1/notebooks/{notebookId}/artifacts
func (h *ArtifactHandler) List(w http.ResponseWriter, r *http.Request) {
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
	artifactType := query.Get("type")
	status := query.Get("status")

	req := &dtos.ListArtifactsRequest{
		NotebookID: notebookID,
		Page:       page,
		Limit:      limit,
		Type:       artifactType,
		Status:     status,
	}

	result, err := h.artifactUseCase.List(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list artifacts: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// respondWithJSON writes a JSON response
func (h *ArtifactHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

// respondWithError writes an error response
func (h *ArtifactHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, dtos.ErrorResponse{
		Code:    http.StatusText(code),
		Message: message,
	})
}
