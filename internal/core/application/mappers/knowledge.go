package mappers

import (
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ToKnowledgeResponse maps a knowledge entity to a response DTO
func ToKnowledgeResponse(knowledge *entities.Knowledge) *dtos.KnowledgeResponse {
	if knowledge == nil {
		return nil
	}

	return &dtos.KnowledgeResponse{
		ID:             knowledge.ID,
		SourceID:       knowledge.SourceID,
		Content:        knowledge.Content,
		ChunkIndex:     knowledge.ChunkIndex,
		HeadingContext: knowledge.HeadingContext,
		FirstPage:      knowledge.FirstPage,
		LastPage:       knowledge.LastPage,
		Metadata:       knowledge.Metadata,
		CreatedAt:      knowledge.CreatedAt,
	}
}

// ToKnowledgeResponses maps a slice of knowledge entities to response DTOs
func ToKnowledgeResponses(knowledges []*entities.Knowledge) []dtos.KnowledgeResponse {
	responses := make([]dtos.KnowledgeResponse, 0, len(knowledges))
	for _, knowledge := range knowledges {
		if knowledge != nil {
			responses = append(responses, *ToKnowledgeResponse(knowledge))
		}
	}
	return responses
}
