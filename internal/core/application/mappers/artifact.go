package mappers

import (
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ToArtifactResponse maps an artifact entity to a response DTO
func ToArtifactResponse(artifact *entities.Artifact) *dtos.ArtifactResponse {
	if artifact == nil {
		return nil
	}

	return &dtos.ArtifactResponse{
		ID:         artifact.ID,
		NotebookID: artifact.NotebookID,
		Title:      artifact.Title,
		Type:       artifact.Type.String(),
		Status:     artifact.Status.String(),
		Format:     artifact.Format.String(),
		Content:    artifact.Content,
		SourceIDs:  artifact.SourceIDs,
		Metadata:   artifact.Metadata,
		Error:      artifact.Error,
		CreatedAt:  artifact.CreatedAt,
		UpdatedAt:  artifact.UpdatedAt,
	}
}

// ToArtifactResponses maps a slice of artifact entities to response DTOs
func ToArtifactResponses(artifacts []*entities.Artifact) []dtos.ArtifactResponse {
	responses := make([]dtos.ArtifactResponse, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact != nil {
			responses = append(responses, *ToArtifactResponse(artifact))
		}
	}
	return responses
}

// ToArtifactListResponses maps a slice of artifact entities to lightweight list DTOs
func ToArtifactListResponses(artifacts []*entities.Artifact) []dtos.ArtifactListResponse {
	responses := make([]dtos.ArtifactListResponse, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact != nil {
			responses = append(responses, dtos.ArtifactListResponse{
				ID:         artifact.ID,
				NotebookID: artifact.NotebookID,
				Title:      artifact.Title,
				Type:       artifact.Type.String(),
				Status:     artifact.Status.String(),
				Format:     artifact.Format.String(),
				CreatedAt:  artifact.CreatedAt,
				UpdatedAt:  artifact.UpdatedAt,
			})
		}
	}
	return responses
}
