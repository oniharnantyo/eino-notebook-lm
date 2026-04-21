package mappers

import (
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ToImageResponse maps an image entity to a response DTO
func ToImageResponse(image *entities.Image) *dtos.ImageResponse {
	if image == nil {
		return nil
	}

	return &dtos.ImageResponse{
		ID:          image.ID,
		SourceID:    image.SourceID,
		S3Key:       image.S3Key,
		Format:      image.Format,
		Width:       image.Width,
		Height:      image.Height,
		Description: image.Description,
		PageNumber:  image.PageNumber,
		Metadata:    image.Metadata,
		CreatedAt:   image.CreatedAt,
	}
}

// ToImageResponses maps a slice of image entities to response DTOs
func ToImageResponses(images []*entities.Image) []dtos.ImageResponse {
	responses := make([]dtos.ImageResponse, 0, len(images))
	for _, image := range images {
		if image != nil {
			responses = append(responses, *ToImageResponse(image))
		}
	}
	return responses
}
