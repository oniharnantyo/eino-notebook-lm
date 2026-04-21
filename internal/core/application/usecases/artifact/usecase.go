package artifact

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// ArtifactUseCase defines the interface for artifact business logic
type ArtifactUseCase interface {
	GetByID(ctx context.Context, id string) (*dtos.ArtifactResponse, error)
	List(ctx context.Context, req *dtos.ListArtifactsRequest) (*dtos.ListArtifactsResponse, error)
}

// artifactUseCase implements ArtifactUseCase
type artifactUseCase struct {
	artifactRepo repositories.ArtifactRepository
}

// NewArtifactUseCase creates a new artifact use case
func NewArtifactUseCase(artifactRepo repositories.ArtifactRepository) ArtifactUseCase {
	return &artifactUseCase{
		artifactRepo: artifactRepo,
	}
}

// GetByID retrieves an artifact by ID
func (uc *artifactUseCase) GetByID(ctx context.Context, id string) (*dtos.ArtifactResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid artifact ID")
	}

	artifact, err := uc.artifactRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.NewInternalError("failed to find artifact", err)
	}
	if artifact == nil {
		return nil, errors.NewNotFoundError("artifact")
	}

	return mappers.ToArtifactResponse(artifact), nil
}

// List retrieves a paginated list of artifacts for a notebook
func (uc *artifactUseCase) List(ctx context.Context, req *dtos.ListArtifactsRequest) (*dtos.ListArtifactsResponse, error) {
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

	filter := repositories.ArtifactFilter{
		NotebookID: &req.NotebookID,
		Limit:      req.Limit,
		Offset:     offset,
		OrderBy:    "created_at",
	}

	if req.Type != "" {
		artifactType := dtos.ParseArtifactType(req.Type)
		filter.Type = &artifactType
	}

	if req.Status != "" {
		status := dtos.ParseArtifactStatus(req.Status)
		filter.Status = &status
	}

	artifacts, total, err := uc.artifactRepo.List(ctx, filter)
	if err != nil {
		return nil, errors.NewInternalError("failed to list artifacts", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListArtifactsResponse{
		Artifacts:  mappers.ToArtifactListResponses(artifacts),
		Total:      int64(total),
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}
