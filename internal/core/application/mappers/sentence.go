package mappers

import (
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ToSentenceResponse maps a sentence entity to a response DTO
func ToSentenceResponse(sentence *entities.Sentence) *dtos.SentenceResponse {
	if sentence == nil {
		return nil
	}

	return &dtos.SentenceResponse{
		ID:          sentence.ID,
		KnowledgeID: sentence.KnowledgeID,
		Content:     sentence.Content,
		Position:    sentence.Position,
		Metadata:    sentence.Metadata,
		CreatedAt:   sentence.CreatedAt,
	}
}

// ToSentenceResponses maps a slice of sentence entities to response DTOs
func ToSentenceResponses(sentences []*entities.Sentence) []dtos.SentenceResponse {
	responses := make([]dtos.SentenceResponse, 0, len(sentences))
	for _, sentence := range sentences {
		if sentence != nil {
			responses = append(responses, *ToSentenceResponse(sentence))
		}
	}
	return responses
}
