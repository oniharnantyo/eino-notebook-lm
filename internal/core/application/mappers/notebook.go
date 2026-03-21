package mappers

import (
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ToNotebookResponse maps a notebook entity to a response DTO
func ToNotebookResponse(notebook *entities.Notebook) *dtos.NotebookResponse {
	if notebook == nil {
		return nil
	}

	return &dtos.NotebookResponse{
		ID:          notebook.ID,
		Title:       notebook.Title,
		Description: notebook.Description,
		Content:     notebook.Content,
		Status:      notebook.Status.String(),
		Tags:        notebook.Tags,
		Metadata:    notebook.Metadata,
		CreatedAt:   notebook.CreatedAt,
		UpdatedAt:   notebook.UpdatedAt,
	}
}

// ToNotebookResponses maps a slice of notebook entities to response DTOs
func ToNotebookResponses(notebooks []*entities.Notebook) []dtos.NotebookResponse {
	responses := make([]dtos.NotebookResponse, 0, len(notebooks))
	for _, notebook := range notebooks {
		if notebook != nil {
			responses = append(responses, *ToNotebookResponse(notebook))
		}
	}
	return responses
}

// ParseID parses a string ID to UUID
func ParseID(id string) (uuid.UUID, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID(""), errors.NewValidationError("invalid ID format")
	}
	return uid, nil
}

// ToEntity maps a create request to an entity (use case logic handles this)
func ToEntity(req *dtos.CreateNotebookRequest) (*entities.Notebook, error) {
	return entities.NewNotebook(req.Title, req.Description, req.Content, req.Tags)
}
