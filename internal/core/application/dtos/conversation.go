package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ConversationResponse represents a conversation response
type ConversationResponse struct {
	ID         string            `json:"id"`
	NotebookID *string           `json:"notebook_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// ListConversationsRequest represents a request to list conversations
type ListConversationsRequest struct {
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	NotebookID string `json:"notebook_id"`
	Model      string `json:"model"`
}

// ListConversationsResponse represents a paginated list of conversations
type ListConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	Total         int64                  `json:"total"`
	Page          int                    `json:"page"`
	Limit         int                    `json:"limit"`
	TotalPages    int                    `json:"total_pages"`
}

// ToConversationResponse maps a conversation entity to a response DTO
func ToConversationResponse(conversation *entities.Conversation) *ConversationResponse {
	if conversation == nil {
		return nil
	}

	return &ConversationResponse{
		ID:         conversation.ID,
		NotebookID: conversation.NotebookID,
		Metadata:   conversation.Metadata,
		CreatedAt:  time.Unix(conversation.CreatedAt, 0),
	}
}

// ToConversationResponses maps a slice of conversation entities to response DTOs
func ToConversationResponses(conversations []*entities.Conversation) []ConversationResponse {
	responses := make([]ConversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		if conversation != nil {
			if resp := ToConversationResponse(conversation); resp != nil {
				responses = append(responses, *resp)
			}
		}
	}
	return responses
}

