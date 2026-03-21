package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// ConversationResponse represents a conversation response
type ConversationResponse struct {
	ID                 string      `json:"id"`
	PreviousResponseID *string     `json:"previous_response_id,omitempty"`
	ResponseID         string      `json:"response_id"`
	RequestInput       interface{} `json:"request_input,omitempty"`
	ResponseMessage    interface{} `json:"response_message,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
}

// ListConversationsRequest represents a request to list conversations
type ListConversationsRequest struct {
	Page               int    `json:"page"`
	Limit              int    `json:"limit"`
	NotebookID         string `json:"notebook_id"`
	Model              string `json:"model"`
	PreviousResponseID string `json:"previous_response_id"`
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
		ID:                 conversation.ID,
		PreviousResponseID: conversation.PreviousResponseID,
		ResponseID:         conversation.ResponseID,
		RequestInput:       conversation.RequestInput,
		ResponseMessage:    conversation.ResponseMessage,
		CreatedAt:          time.Unix(conversation.CreatedAt, 0),
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
