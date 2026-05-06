package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/knowledge"
	sourceUseCase "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/source"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// KnowledgeHandler handles knowledge HTTP requests
type KnowledgeHandler struct {
	useCase       knowledge.KnowledgeUseCase
	sourceUseCase sourceUseCase.SourceUseCase
	logger        *logger.Logger
}

// NewKnowledgeHandler creates a new knowledge handler
func NewKnowledgeHandler(
	useCase knowledge.KnowledgeUseCase,
	sourceUseCase sourceUseCase.SourceUseCase,
	log *logger.Logger,
) *KnowledgeHandler {
	return &KnowledgeHandler{
		useCase:       useCase,
		sourceUseCase: sourceUseCase,
		logger:        log,
	}
}

// Create handles knowledge creation requests via multipart/form-data
// Supports: file uploads (PDF, markdown, images), URLs, and direct text content
// Supports async processing via "async" form field
func (h *KnowledgeHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 100MB
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	// Extract notebookId from URL path
	vars := mux.Vars(r)
	notebookIDStr := vars["notebookId"]
	if notebookIDStr == "" {
		h.respondWithError(w, http.StatusBadRequest, "notebookId is required in URL path")
		return
	}

	notebookID, err := mappers.ParseID(notebookIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "invalid notebook_id format")
		return
	}

	// CRITICAL: Parse multipart form ONCE before accessing any form data
	// This ensures both files and form values are accessible
	// Using 32MB max memory for in-memory storage, rest spills to disk
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse multipart form: %v", err))
		return
	}

	// Parse form values
	title := r.FormValue("title")
	contentText := r.FormValue("content")
	url := r.FormValue("url")
	sourceType := r.FormValue("source_type")
	metadataStr := r.FormValue("metadata")
	subIndexesStr := r.FormValue("sub_indexes")
	asyncStr := r.FormValue("async")
	async := asyncStr == "true" || asyncStr == "1"

	// Parse metadata if provided
	var metadata map[string]interface{}
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
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

	// Determine content type and build request
	ingestReq, err := h.buildIngestRequest(r, notebookID, title, contentText, url, sourceType, metadata, subIndexes, async)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Call usecase to handle ingestion
	response, _, err := h.sourceUseCase.IngestContent(r.Context(), ingestReq)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return appropriate response based on async/sync
	if response.IsAsync {
		h.respondWithJSON(w, http.StatusAccepted, response)
		return
	}

	h.respondWithJSON(w, http.StatusCreated, response)
}

func (h *KnowledgeHandler) StreamSourceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sourceIDStr := vars["sourceId"]

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.respondWithError(w, http.StatusNotImplemented, "streaming not supported")
		return
	}

	// Poll database every 500ms until terminal state
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastStatus entities.SourceStatus
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			statusResp, err := h.sourceUseCase.GetStatus(r.Context(), sourceIDStr)
			if err != nil {
				h.logger.Error("Failed to get source status", "source_id", sourceIDStr, "error", err)
				return
			}

			currentStatus := entities.SourceStatus(statusResp.Status)

			// Send event only if status changed
			if currentStatus != lastStatus {
				h.sendSSEEvent(w, flusher, statusResp)
				lastStatus = currentStatus
			}

			// Close connection on terminal state
			if currentStatus == entities.SourceStatusCompleted || currentStatus == entities.SourceStatusFailed {
				return
			}
		}
	}
}

// sendSSEEvent sends an SSE event with source status
func (h *KnowledgeHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, statusResp *dtos.KnowledgeIngestionStatusResponse) {
	event := map[string]interface{}{
		"source_id":  statusResp.SourceID,
		"status":     statusResp.Status,
		"error":      statusResp.Error,
		"updated_at": statusResp.UpdatedAt,
	}

	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal SSE event", "error", err)
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// buildIngestRequest builds an IngestContentRequest from the HTTP request
func (h *KnowledgeHandler) buildIngestRequest(
	r *http.Request,
	notebookID uuid.UUID,
	title, contentText, url, sourceType string,
	metadata map[string]interface{},
	subIndexes []string,
	async bool,
) (*dtos.IngestContentRequest, error) {
	// Check for file upload
	file, header, err := r.FormFile("file")
	if err == nil && header != nil {
		// Determine MIME type from file extension
		mimeType := h.detectMimeType(header.Filename)

		// Set title if not provided
		if title == "" {
			title = header.Filename
		}

		return &dtos.IngestContentRequest{
			NotebookID:  notebookID,
			Title:       title,
			URI:         url,
			ContentType: usecases.ContentTypeFile,
			MIMEType:    mimeType,
			SourceType:  sourceType,
			Metadata:    metadata,
			SubIndexes:  subIndexes,
			FileContent: file,
			Filename:    header.Filename,
			Async:       async,
		}, nil
	}

	// Check for URL
	if url != "" {
		// Set title if not provided
		if title == "" {
			title = url
		}

		return &dtos.IngestContentRequest{
			NotebookID:  notebookID,
			Title:       title,
			URI:         url,
			ContentType: usecases.ContentTypeURL,
			MIMEType:    entities.ContentTypeWebsite,
			SourceType:  sourceType,
			Metadata:    metadata,
			SubIndexes:  subIndexes,
			Async:       async,
		}, nil
	}

	// Check for direct text content
	if contentText != "" {
		// Set title if not provided
		if title == "" {
			title = "Untitled"
		}

		return &dtos.IngestContentRequest{
			NotebookID:  notebookID,
			Title:       title,
			ContentType: usecases.ContentTypeText,
			MIMEType:    entities.ContentTypeText,
			SourceType:  sourceType,
			Metadata:    metadata,
			SubIndexes:  subIndexes,
			TextContent: contentText,
			Async:       async,
		}, nil
	}

	return nil, fmt.Errorf("either file, url, or content must be provided")
}

// detectMimeType detects MIME type from filename extension
func (h *KnowledgeHandler) detectMimeType(filename string) entities.ContentType {
	// Simple extension-based detection
	// In production, use http.DetectContentType or a more sophisticated library
	switch {
	case endsWith(filename, ".pdf"):
		return entities.ContentTypePDF
	case endsWith(filename, ".docx"):
		return entities.ContentTypeDocx
	case endsWith(filename, ".md"):
		return entities.ContentTypeMarkdown
	case endsWith(filename, ".txt"):
		return entities.ContentTypeText
	case endsWith(filename, ".html") || endsWith(filename, ".htm"):
		return entities.ContentTypeWebsite
	default:
		return entities.ContentTypeOther
	}
}

// endsWith checks if a string ends with a suffix (case-insensitive)
func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
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
