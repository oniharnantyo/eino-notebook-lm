package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// CreateKnowledgeRequest represents a request to create knowledge
type CreateKnowledgeRequest struct {
	NotebookID uuid.UUID                      `json:"notebook_id" validate:"required"`
	Title      string                         `json:"title"`
	Content    string                         `json:"content" validate:"required"`
	SourceType string                         `json:"source_type" validate:"omitempty,oneof=document website text api other"`
	Metadata   map[string]interface{}         `json:"metadata,omitempty"`
	SubIndexes []string                       `json:"sub_indexes,omitempty"`
}

// CreateKnowledgeMultipartRequest represents a multipart request to create knowledge
// Supports: file uploads (PDF, images, etc.), URLs, and direct text/markdown content
type CreateKnowledgeMultipartRequest struct {
	NotebookID  string                `form:"notebook_id" validate:"required"`
	Title       string                `form:"title"`
	Content     string                `form:"content"`        // For text/markdown input
	URL         string                `form:"url"`            // For website content
	SourceType  string                `form:"source_type"`    // document, website, text, api, other
	Metadata    string                `form:"metadata"`       // JSON string for metadata
	SubIndexes  string                `form:"sub_indexes"`    // JSON array for sub_indexes
	File        *MultipartFileHeader  `form:"file"`           // For file uploads (PDF, etc.)
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
	KnowledgeID uuid.UUID                      `json:"knowledge_id" validate:"required"`
	Title       string                         `json:"title"`
	Content     string                         `json:"content"`
	SourceType  string                         `json:"source_type" validate:"omitempty,oneof=document website text api other"`
	Metadata    map[string]interface{}         `json:"metadata,omitempty"`
	SubIndexes  []string                       `json:"sub_indexes,omitempty"`
}

// KnowledgeResponse represents a knowledge response
type KnowledgeResponse struct {
	KnowledgeID uuid.UUID                      `json:"knowledge_id"`
	NotebookID  uuid.UUID                      `json:"notebook_id"`
	Title       string                         `json:"title"`
	Content     string                         `json:"content"`
	SourceType  string                         `json:"source_type"`
	Metadata    map[string]interface{}         `json:"metadata,omitempty"`
	SubIndexes  []string                       `json:"sub_indexes,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
}

// ListKnowledgesRequest represents a request to list knowledges
type ListKnowledgesRequest struct {
	NotebookID  uuid.UUID `json:"notebook_id" validate:"required"`
	Page        int       `json:"page" validate:"min=1"`
	Limit       int       `json:"limit" validate:"min=1,max=100"`
	SourceType  string    `json:"source_type" validate:"omitempty,oneof=document website text api other"`
	Query       string    `json:"query" validate:"max=100"`
}

// ListKnowledgesResponse represents a paginated list of knowledges
type ListKnowledgesResponse struct {
	Knowledges  []KnowledgeResponse `json:"knowledges"`
	Total       int64              `json:"total"`
	Page        int                `json:"page"`
	Limit       int                `json:"limit"`
	TotalPages  int                `json:"total_pages"`
}

// ToKnowledgeResponse maps a knowledge entity to a response DTO
func ToKnowledgeResponse(knowledge *entities.Knowledge) *KnowledgeResponse {
	if knowledge == nil {
		return nil
	}

	return &KnowledgeResponse{
		KnowledgeID: knowledge.KnowledgeID,
		NotebookID:  knowledge.NotebookID,
		Title:       knowledge.Title,
		Content:     knowledge.Content,
		SourceType:  string(knowledge.SourceType),
		Metadata:    knowledge.Metadata,
		SubIndexes:  knowledge.SubIndexes,
		CreatedAt:   knowledge.CreatedAt,
	}
}

// ToKnowledgeResponses maps a slice of knowledge entities to response DTOs
func ToKnowledgeResponses(knowledges []*entities.Knowledge) []KnowledgeResponse {
	responses := make([]KnowledgeResponse, 0, len(knowledges))
	for _, knowledge := range knowledges {
		if knowledge != nil {
			responses = append(responses, *ToKnowledgeResponse(knowledge))
		}
	}
	return responses
}

// ParseSourceType parses a string to KnowledgeSource
func ParseSourceType(sourceType string) entities.KnowledgeSource {
	switch sourceType {
	case "document":
		return entities.SourceDocument
	case "website":
		return entities.SourceWebsite
	case "text":
		return entities.SourceText
	case "api":
		return entities.SourceAPI
	case "other":
		return entities.SourceOther
	default:
		return entities.SourceDocument
	}
}
