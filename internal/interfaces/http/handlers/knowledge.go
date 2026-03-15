package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/knowledge"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
)

// KnowledgeHandler handles knowledge HTTP requests
type KnowledgeHandler struct {
	useCase                 knowledge.KnowledgeUseCase
	notebookRepo            repositories.NotebookRepository
	contentExtractorFactory extractor.ContentExtractorFactory
	documentParserFactory   *document.DocumentParserFactory
	logger                  *logger.Logger
}

// NewKnowledgeHandler creates a new knowledge handler
// Dependency Inversion: Depends on ContentExtractorFactory abstraction
func NewKnowledgeHandler(
	useCase knowledge.KnowledgeUseCase,
	notebookRepo repositories.NotebookRepository,
	contentExtractorFactory extractor.ContentExtractorFactory,
	documentParserFactory *document.DocumentParserFactory,
	log *logger.Logger,
) *KnowledgeHandler {
	return &KnowledgeHandler{
		useCase:                 useCase,
		notebookRepo:            notebookRepo,
		contentExtractorFactory: contentExtractorFactory,
		documentParserFactory:   documentParserFactory,
		logger:                  log,
	}
}

// Create handles knowledge creation requests via multipart/form-data
// Supports: file uploads (PDF, markdown, images), URLs, and direct text content
// Uses Strategy Pattern via ContentExtractor for different content types
func (h *KnowledgeHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 100MB
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)


	// Extract form fields
	notebookIDStr := r.FormValue("notebook_id")
	if notebookIDStr == "" {
		h.respondWithError(w, http.StatusBadRequest, "notebook_id is required")
		return
	}

	notebookID, err := mappers.ParseID(notebookIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid notebook_id format")
		return
	}

	// Validate notebook exists before proceeding
	exists, err := h.notebookRepo.Exists(r.Context(), notebookID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to validate notebook: %v", err))
		return
	}
	if !exists {
		h.respondWithError(w, http.StatusNotFound, fmt.Sprintf("Notebook with ID %s not found", notebookIDStr))
		return
	}

	title := r.FormValue("title")
	contentText := r.FormValue("content")
	url := r.FormValue("url")
	sourceType := r.FormValue("source_type")
	metadataStr := r.FormValue("metadata")
	subIndexesStr := r.FormValue("sub_indexes")

	// Parse metadata if provided
	var baseMetadata map[string]interface{}
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &baseMetadata); err != nil {
			h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid metadata JSON: %v", err))
			return
		}
	}

	// Parse sub_indexes if provided
	var subIndexes []string
	if subIndexesStr != "" {
		if err := json.Unmarshal([]byte(subIndexesStr), &subIndexes); err != nil {
			h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid sub_indexes JSON: %v", err))
			return
		}
	}

	// Determine content type and create content source
	contentType, contentSource, err := h.buildContentSource(r, contentText, url, baseMetadata)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Get appropriate extractor using factory
	contentExtractor, err := h.contentExtractorFactory.GetExtractor(contentType)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Unsupported content type: %v", err))
		return
	}

	// Extract content using strategy pattern
	extractedContent, extractedMetadata, err := contentExtractor.Extract(r.Context(), contentSource)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to extract content: %v", err))
		return
	}

	// Merge metadata
	metadata := extractedMetadata
	if baseMetadata != nil {
		for k, v := range baseMetadata {
			metadata[k] = v
		}
	}

	// Set title if not provided
	if title == "" {
		if contentType == usecases.ContentTypeFile && contentSource.Filename != "" {
			title = contentSource.Filename
		} else if contentType == usecases.ContentTypeURL && url != "" {
			title = url
		} else {
			title = "Untitled"
		}
	}

	// Determine source type
	finalSourceType := sourceType
	if finalSourceType == "" {
		finalSourceType = string(contentType)
	}

	// Create the request
	req := &dtos.CreateKnowledgeRequest{
		NotebookID: notebookID,
		Title:      title,
		Content:    extractedContent,
		SourceType: finalSourceType,
		Metadata:   metadata,
		SubIndexes: subIndexes,
	}

	// Call use case
	knowledge, err := h.useCase.Create(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create knowledge: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusCreated, knowledge)
}

// buildContentSource builds a ContentSource from the request
// Open/Closed Principle: Easy to extend for new content types
func (h *KnowledgeHandler) buildContentSource(
	r *http.Request,
	contentText string,
	url string,
	metadata map[string]interface{},
) (usecases.ContentType, usecases.ContentSource, error) {
	// Check for file upload
	file, header, err := r.FormFile("file")
	if err == nil && header != nil {
		// Read file content into memory to avoid "file already closed" error
		content, err := io.ReadAll(file)
		file.Close() // Close immediately after reading
		if err != nil {
			return "", usecases.ContentSource{}, fmt.Errorf("failed to read file: %w", err)
		}

		// Return ContentSource with bytes.Reader instead of the multipart file
		return usecases.ContentTypeFile, usecases.ContentSource{
			Type:     usecases.ContentTypeFile,
			Reader:   bytes.NewReader(content),
			Filename: header.Filename,
			Metadata: metadata,
		}, nil
	}

	// Check for URL
	if url != "" {
		return usecases.ContentTypeURL, usecases.ContentSource{
			Type:     usecases.ContentTypeURL,
			URL:      url,
			Metadata: metadata,
		}, nil
	}

	// Check for direct text content
	if contentText != "" {
		return usecases.ContentTypeText, usecases.ContentSource{
			Type:     usecases.ContentTypeText,
			Text:     contentText,
			Metadata: metadata,
		}, nil
	}

	return "", usecases.ContentSource{}, fmt.Errorf("either file, url, or content must be provided")
}

// GetByID handles get knowledge by ID requests
func (h *KnowledgeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	knowledge, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		h.respondWithError(w, http.StatusNotFound, fmt.Sprintf("Knowledge not found: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, knowledge)
}

// List handles list knowledges requests
func (h *KnowledgeHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	notebookIDStr := query.Get("notebook_id")
	if notebookIDStr == "" {
		h.respondWithError(w, http.StatusBadRequest, "notebook_id is required")
		return
	}

	notebookID, err := mappers.ParseID(notebookIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid notebook_id format")
		return
	}

	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))
	sourceType := query.Get("source_type")
	searchQuery := query.Get("q")

	req := &dtos.ListKnowledgesRequest{
		NotebookID: notebookID,
		Page:       page,
		Limit:      limit,
		SourceType: sourceType,
		Query:      searchQuery,
	}

	result, err := h.useCase.List(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list knowledges: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// Update handles knowledge update requests
func (h *KnowledgeHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	knowledgeID, err := mappers.ParseID(idStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid knowledge ID format")
		return
	}

	var req dtos.UpdateKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Set ID from URL
	req.KnowledgeID = knowledgeID

	knowledge, err := h.useCase.Update(r.Context(), &req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update knowledge: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, knowledge)
}

// Delete handles knowledge deletion requests
func (h *KnowledgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.useCase.Delete(r.Context(), id); err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete knowledge: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Search handles knowledge search requests
func (h *KnowledgeHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.respondWithError(w, http.StatusBadRequest, "search query is required")
		return
	}

	notebookIDStr := r.URL.Query().Get("notebook_id")
	if notebookIDStr == "" {
		h.respondWithError(w, http.StatusBadRequest, "notebook_id is required")
		return
	}

	notebookID, err := mappers.ParseID(notebookIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid notebook_id format")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	req := &dtos.ListKnowledgesRequest{
		NotebookID: notebookID,
		Query:      query,
		Page:       page,
		Limit:      limit,
	}

	result, err := h.useCase.Search(r.Context(), req)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to search knowledges: %v", err))
		return
	}

	h.respondWithJSON(w, http.StatusOK, result)
}

// respondWithJSON writes a JSON response
func (h *KnowledgeHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError writes an error response
func (h *KnowledgeHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, dtos.ErrorResponse{
		Code:    http.StatusText(code),
		Message: message,
	})
}
