package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// CreateDocumentRequest represents a request to create a document
// TODO: Implement request DTO based on requirements
type CreateDocumentRequest struct {
	Title   string                 `json:"title"`
	Content string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateDocumentRequest represents a request to update a document
// TODO: Implement request DTO based on requirements
type UpdateDocumentRequest struct {
	ID       uuid.UUID               `json:"id"`
	Title    string                  `json:"title"`
	Content  string                  `json:"content"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`
}

// DocumentResponse represents a document response
// TODO: Implement response DTO based on requirements
type DocumentResponse struct {
	ID        uuid.UUID              `json:"id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ListDocumentsRequest represents a request to list documents
// TODO: Implement request DTO based on requirements
type ListDocumentsRequest struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// ListDocumentsResponse represents a paginated list of documents
// TODO: Implement response DTO based on requirements
type ListDocumentsResponse struct {
	Documents  []DocumentResponse `json:"documents"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}
