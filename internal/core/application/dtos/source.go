package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// CreateSourceRequest represents a request to create a source
type CreateSourceRequest struct {
	NotebookID  uuid.UUID                      `json:"notebook_id" validate:"required"`
	Title       string                         `json:"title" validate:"required"`
	URI         string                         `json:"uri,omitempty"`
	ContentType string                         `json:"content_type" validate:"required"`
	Metadata    map[string]any                 `json:"metadata,omitempty"`
}

// SourceResponse represents a source response
type SourceResponse struct {
	ID          uuid.UUID                      `json:"id"`
	NotebookID  uuid.UUID                      `json:"notebook_id"`
	Title       string                         `json:"title"`
	URI         string                         `json:"uri,omitempty"`
	ContentType string                         `json:"content_type"`
	Content     string                         `json:"content,omitempty"`
	ChunkCount  int                            `json:"chunk_count"`
	TotalSize   int                            `json:"total_size,omitempty"`
	Metadata    map[string]any                 `json:"metadata,omitempty"`
	Status      string                         `json:"status"`
	Error       *string                        `json:"error,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
}

// ListSourcesRequest represents a request to list sources
type ListSourcesRequest struct {
	NotebookID  uuid.UUID `json:"notebook_id" validate:"required"`
	Page        int       `json:"page" validate:"min=1"`
	Limit       int       `json:"limit" validate:"min=1,max=100"`
	ContentType string    `json:"content_type" validate:"omitempty"`
}

// SourceListResponse represents a lightweight source for list responses
type SourceListResponse struct {
	ID          uuid.UUID `json:"id"`
	NotebookID  uuid.UUID `json:"notebook_id"`
	Title       string    `json:"title"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListSourcesResponse represents a paginated list of sources
type ListSourcesResponse struct {
	Sources    []SourceListResponse `json:"sources"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

// ToSourceResponse maps a source entity to a response DTO
func ToSourceResponse(source *entities.Source) *SourceResponse {
	if source == nil {
		return nil
	}

	return &SourceResponse{
		ID:          source.ID,
		NotebookID:  source.NotebookID,
		Title:       source.Title,
		URI:         source.URI,
		ContentType: string(source.ContentType),
		Content:     source.Content,
		ChunkCount:  source.ChunkCount,
		TotalSize:   source.TotalSize,
		Metadata:    source.Metadata,
		Status:      source.Status,
		Error:       source.Error,
		CreatedAt:   source.CreatedAt,
		UpdatedAt:   source.UpdatedAt,
	}
}

// ToSourceResponses maps a slice of source entities to response DTOs
func ToSourceResponses(sources []*entities.Source) []SourceResponse {
	responses := make([]SourceResponse, 0, len(sources))
	for _, source := range sources {
		if source != nil {
			responses = append(responses, *ToSourceResponse(source))
		}
	}
	return responses
}

// ToSourceListResponses maps a slice of source entities to lightweight list DTOs
func ToSourceListResponses(sources []*entities.Source) []SourceListResponse {
	responses := make([]SourceListResponse, 0, len(sources))
	for _, source := range sources {
		if source != nil {
			responses = append(responses, SourceListResponse{
				ID:          source.ID,
				NotebookID:  source.NotebookID,
				Title:       source.Title,
				ContentType: string(source.ContentType),
				CreatedAt:   source.CreatedAt,
				UpdatedAt:   source.UpdatedAt,
			})
		}
	}
	return responses
}

// ParseContentType parses a string to ContentType
func ParseContentType(contentType string) entities.ContentType {
	switch contentType {
	case "application/pdf":
		return entities.ContentTypePDF
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return entities.ContentTypeDocx
	case "text/html":
		return entities.ContentTypeWebsite
	case "text/plain":
		return entities.ContentTypeText
	case "text/markdown":
		return entities.ContentTypeMarkdown
	case "application/json":
		return entities.ContentTypeAPI
	default:
		return entities.ContentTypeOther
	}
}
