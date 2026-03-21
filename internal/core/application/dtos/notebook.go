package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// CreateNotebookRequest represents a request to create a notebook
type CreateNotebookRequest struct {
	Title       string    `json:"title" validate:"required,min=1,max=200"`
	Description string    `json:"description" validate:"max=500"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags" validate:"max=10"`
}

// UpdateNotebookRequest represents a request to update a notebook
type UpdateNotebookRequest struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Title       string    `json:"title" validate:"required,min=1,max=200"`
	Description string    `json:"description" validate:"max=500"`
	Content     string    `json:"content"`
	Tags        []string  `json:"tags" validate:"max=10"`
}

// NotebookResponse represents a notebook response
type NotebookResponse struct {
	ID          uuid.UUID              `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Content     string                 `json:"content"`
	Status      string                 `json:"status"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ListNotebooksRequest represents a request to list notebooks
type ListNotebooksRequest struct {
	Page   int       `json:"page" validate:"min=1"`
	Limit  int       `json:"limit" validate:"min=1,max=100"`
	Status string    `json:"status" validate:"omitempty,oneof=active archived deleted"`
	Tags   []string  `json:"tags" validate:"max=5"`
	Query  string    `json:"query" validate:"max=100"`
}

// ListNotebooksResponse represents a paginated list of notebooks
type ListNotebooksResponse struct {
	Notebooks []NotebookResponse `json:"notebooks"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
