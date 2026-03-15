package notebook

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
)

// NotebookUseCase defines the interface for notebook business logic
type NotebookUseCase interface {
	Create(ctx context.Context, req *dtos.CreateNotebookRequest) (*dtos.NotebookResponse, error)
	GetByID(ctx context.Context, id string) (*dtos.NotebookResponse, error)
	List(ctx context.Context, req *dtos.ListNotebooksRequest) (*dtos.ListNotebooksResponse, error)
	Update(ctx context.Context, req *dtos.UpdateNotebookRequest) (*dtos.NotebookResponse, error)
	Delete(ctx context.Context, id string) error
	Archive(ctx context.Context, id string) error
	Search(ctx context.Context, query string, page, limit int) (*dtos.ListNotebooksResponse, error)
}

// notebookUseCase implements NotebookUseCase
type notebookUseCase struct {
	notebookRepo repositories.NotebookRepository
}

// NewNotebookUseCase creates a new notebook use case
func NewNotebookUseCase(notebookRepo repositories.NotebookRepository) NotebookUseCase {
	return &notebookUseCase{
		notebookRepo: notebookRepo,
	}
}

// Create creates a new notebook
func (uc *notebookUseCase) Create(ctx context.Context, req *dtos.CreateNotebookRequest) (*dtos.NotebookResponse, error) {
	// Create the entity
	notebook, err := mappers.ToEntity(req)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create notebook: %v", err))
	}

	// Save to repository
	if err := uc.notebookRepo.Save(ctx, notebook); err != nil {
		return nil, errors.NewInternalError("failed to save notebook", err)
	}

	// Map to response
	return mappers.ToNotebookResponse(notebook), nil
}

// GetByID retrieves a notebook by ID
func (uc *notebookUseCase) GetByID(ctx context.Context, id string) (*dtos.NotebookResponse, error) {
	// Parse ID
	uid, err := mappers.ParseID(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid notebook ID")
	}

	// Find by ID
	notebook, err := uc.notebookRepo.FindByID(ctx, uid)
	if err != nil {
		return nil, errors.NewInternalError("failed to find notebook", err)
	}
	if notebook == nil {
		return nil, errors.NewNotFoundError("notebook")
	}

	return mappers.ToNotebookResponse(notebook), nil
}

// List retrieves a paginated list of notebooks
func (uc *notebookUseCase) List(ctx context.Context, req *dtos.ListNotebooksRequest) (*dtos.ListNotebooksResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	offset := (req.Page - 1) * req.Limit

	var notebooks []*entities.Notebook
	var err error
	var total int64

	// Filter by user_id first, then by status or tags if provided
	switch {
	case req.Status != "":
		notebooks, err = uc.notebookRepo.FindByStatus(ctx, req.Status, req.Limit, offset)
		total, _ = uc.notebookRepo.CountByUserID(ctx, req.UserID)
	case len(req.Tags) > 0:
		notebooks, err = uc.notebookRepo.FindByTags(ctx, req.Tags, req.Limit, offset)
		total, _ = uc.notebookRepo.CountByUserID(ctx, req.UserID)
	case req.Query != "":
		notebooks, err = uc.notebookRepo.Search(ctx, req.Query, req.Limit, offset)
		total, _ = uc.notebookRepo.CountByUserID(ctx, req.UserID)
	default:
		notebooks, err = uc.notebookRepo.FindByUserID(ctx, req.UserID, req.Limit, offset)
		total, err = uc.notebookRepo.CountByUserID(ctx, req.UserID)
	}

	if err != nil {
		return nil, errors.NewInternalError("failed to list notebooks", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListNotebooksResponse{
		Notebooks: mappers.ToNotebookResponses(notebooks),
		Total:     total,
		Page:      req.Page,
		Limit:     req.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing notebook
func (uc *notebookUseCase) Update(ctx context.Context, req *dtos.UpdateNotebookRequest) (*dtos.NotebookResponse, error) {
	// Check if notebook exists
	notebook, err := uc.notebookRepo.FindByID(ctx, req.ID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find notebook", err)
	}
	if notebook == nil {
		return nil, errors.NewNotFoundError("notebook")
	}

	// Update the entity
	if err := notebook.Update(req.Title, req.Description, req.Content, req.Tags); err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("failed to update notebook: %v", err))
	}

	// Save to repository
	if err := uc.notebookRepo.Save(ctx, notebook); err != nil {
		return nil, errors.NewInternalError("failed to save notebook", err)
	}

	return mappers.ToNotebookResponse(notebook), nil
}

// Delete deletes a notebook by ID
func (uc *notebookUseCase) Delete(ctx context.Context, id string) error {
	uid, err := mappers.ParseID(id)
	if err != nil {
		return errors.NewValidationError("invalid notebook ID")
	}

	return uc.notebookRepo.Delete(ctx, uid)
}

// Archive archives a notebook
func (uc *notebookUseCase) Archive(ctx context.Context, id string) error {
	uid, err := mappers.ParseID(id)
	if err != nil {
		return errors.NewValidationError("invalid notebook ID")
	}

	notebook, err := uc.notebookRepo.FindByID(ctx, uid)
	if err != nil {
		return errors.NewInternalError("failed to find notebook", err)
	}
	if notebook == nil {
		return errors.NewNotFoundError("notebook")
	}

	notebook.Archive()
	return uc.notebookRepo.Save(ctx, notebook)
}

// Search searches notebooks by query
func (uc *notebookUseCase) Search(ctx context.Context, query string, page, limit int) (*dtos.ListNotebooksResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	notebooks, err := uc.notebookRepo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, errors.NewInternalError("failed to search notebooks", err)
	}

	total, _ := uc.notebookRepo.Count(ctx)
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return &dtos.ListNotebooksResponse{
		Notebooks: mappers.ToNotebookResponses(notebooks),
		Total:     total,
		Page:      page,
		Limit:     limit,
		TotalPages: totalPages,
	}, nil
}
