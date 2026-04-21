package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// CreateKnowledgeRequest represents a request to create knowledge
type CreateKnowledgeRequest struct {
	SourceID       uuid.UUID      `json:"source_id" validate:"required"`
	Content        string         `json:"content" validate:"required"`
	ChunkIndex     int            `json:"chunk_index"`
	HeadingContext map[string]any `json:"heading_context,omitempty"`
	FirstPage      int            `json:"first_page,omitempty"`
	LastPage       int            `json:"last_page,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// CreateKnowledgeMultipartRequest represents a multipart request to create knowledge
// Supports: file uploads (PDF, images, etc.), URLs, and direct text/markdown content
type CreateKnowledgeMultipartRequest struct {
	NotebookID string               `form:"notebook_id" validate:"required"`
	Title      string               `form:"title"`
	Content    string               `form:"content"`  // For text/markdown input
	URL        string               `form:"url"`      // For website content
	Metadata   string               `form:"metadata"` // JSON string for metadata
	File       *MultipartFileHeader `form:"file"`     // For file uploads (PDF, etc.)
}

// MultipartFileHeader represents an uploaded file in multipart form
type MultipartFileHeader struct {
	Filename string
	Size     int64
	Content  []byte
	Header   map[string][]string
}

// UpdateKnowledgeRequest represents a request to update knowledge
type UpdateKnowledgeRequest struct {
	ID             uuid.UUID              `json:"id" validate:"required"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	HeadingContext map[string]any         `json:"heading_context,omitempty"`
}

// KnowledgeResponse represents a knowledge response
type KnowledgeResponse struct {
	ID             uuid.UUID      `json:"id"`
	SourceID       uuid.UUID      `json:"source_id"`
	Content        string         `json:"content"`
	ChunkIndex     int            `json:"chunk_index"`
	HeadingContext map[string]any `json:"heading_context,omitempty"`
	FirstPage      int            `json:"first_page,omitempty"`
	LastPage       int            `json:"last_page,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ListKnowledgesRequest represents a request to list knowledges
type ListKnowledgesRequest struct {
	SourceID uuid.UUID `json:"source_id" validate:"required"`
	Page     int       `json:"page" validate:"min=1"`
	Limit    int       `json:"limit" validate:"min=1,max=100"`
	Query    string    `json:"query" validate:"max=100"`
}

// ListKnowledgesResponse represents a paginated list of knowledges
type ListKnowledgesResponse struct {
	Knowledges []KnowledgeResponse `json:"knowledges"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}

// AsyncKnowledgeResponse represents an async knowledge ingestion response
type AsyncKnowledgeResponse struct {
	SourceID        uuid.UUID `json:"source_id"`
	Status          string    `json:"status"`
	StatusURL       string    `json:"status_url"`
	StatusStreamURL string    `json:"status_stream_url"`
}

// KnowledgeIngestionStatusResponse represents the status of a knowledge ingestion source
type KnowledgeIngestionStatusResponse struct {
	SourceID  uuid.UUID `json:"source_id"`
	Status    string    `json:"status"`
	Error     *string   `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
