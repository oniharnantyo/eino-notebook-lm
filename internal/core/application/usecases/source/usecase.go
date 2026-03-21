package source

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SourceUseCase defines the interface for source business logic
type SourceUseCase interface {
	Create(ctx context.Context, req *dtos.CreateSourceRequest) (*dtos.SourceResponse, error)
	GetByID(ctx context.Context, id string) (*dtos.SourceResponse, error)
	List(ctx context.Context, req *dtos.ListSourcesRequest) (*dtos.ListSourcesResponse, error)
	Update(ctx context.Context, sourceID uuid.UUID, content string, size int) error
	Delete(ctx context.Context, id string) error
}

// sourceUseCase implements SourceUseCase
type sourceUseCase struct {
	sourceRepo   repositories.SourceRepository
	notebookRepo repositories.NotebookRepository
}

// NewSourceUseCase creates a new source use case
func NewSourceUseCase(
	sourceRepo repositories.SourceRepository,
	notebookRepo repositories.NotebookRepository,
) SourceUseCase {
	return &sourceUseCase{
		sourceRepo:   sourceRepo,
		notebookRepo: notebookRepo,
	}
}

// Create creates a new source
func (uc *sourceUseCase) Create(ctx context.Context, req *dtos.CreateSourceRequest) (*dtos.SourceResponse, error) {
	// Verify notebook exists
	_, err := uc.notebookRepo.FindByID(ctx, req.NotebookID)
	if err != nil {
		return nil, errors.NewInternalError("failed to validate notebook", err)
	}

	// Check for duplicate URI if provided
	if req.URI != "" {
		existing, _ := uc.sourceRepo.GetByURI(ctx, req.NotebookID, req.URI)
		if existing != nil {
			return nil, errors.NewValidationError("source with this URI already exists")
		}
	}

	// Parse content type
	contentType := dtos.ParseContentType(req.ContentType)

	// Create source
	source, err := entities.NewSource(req.NotebookID, req.Title, req.URI, contentType)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create source: %v", err))
	}

	// Set metadata if provided
	if req.Metadata != nil {
		source.Metadata = req.Metadata
	}

	// Save to repository
	if err := uc.sourceRepo.Create(ctx, source); err != nil {
		return nil, errors.NewInternalError("failed to save source", err)
	}

	return dtos.ToSourceResponse(source), nil
}

// GetByID retrieves a source by ID
func (uc *sourceUseCase) GetByID(ctx context.Context, id string) (*dtos.SourceResponse, error) {
	// Parse ID
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid source ID")
	}

	// Find by ID
	source, err := uc.sourceRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return nil, errors.NewNotFoundError("source")
	}

	return dtos.ToSourceResponse(source), nil
}

// List retrieves a paginated list of sources for a notebook
func (uc *sourceUseCase) List(ctx context.Context, req *dtos.ListSourcesRequest) (*dtos.ListSourcesResponse, error) {
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

	// Build filter
	filter := repositories.SourceFilter{
		NotebookID: &req.NotebookID,
		Limit:      req.Limit,
		Offset:     offset,
		OrderBy:    "created_at",
	}

	if req.ContentType != "" {
		ct := dtos.ParseContentType(req.ContentType)
		filter.ContentType = &ct
	}

	// Get sources
	sources, total, err := uc.sourceRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewInternalError("failed to list sources", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListSourcesResponse{
		Sources:    dtos.ToSourceListResponses(sources),
		Total:      int64(total),
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates source content
func (uc *sourceUseCase) Update(ctx context.Context, sourceID uuid.UUID, content string, size int) error {
	source, err := uc.sourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return errors.NewNotFoundError("source")
	}

	source.SetContent(content, size)

	if err := uc.sourceRepo.Update(ctx, source); err != nil {
		return errors.NewInternalError("failed to update source", err)
	}

	return nil
}

// Delete deletes a source by ID
func (uc *sourceUseCase) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid source ID")
	}

	return uc.sourceRepo.Delete(ctx, uid)
}
