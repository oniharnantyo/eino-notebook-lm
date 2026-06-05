package dtos

import (
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
)

// MessageResponse represents a message response
type MessageResponse struct {
	ID                 string         `json:"id"`
	ConversationID     string         `json:"conversation_id"`
	SequenceNum        int            `json:"sequence_num"`
	ResponseID         string         `json:"response_id"`
	PreviousResponseID *string        `json:"previous_response_id,omitempty"`
	Message            map[string]any `json:"message"`
	Model              string         `json:"model,omitempty"`
	FinishReason       string         `json:"finish_reason,omitempty"`
	PromptTokens       int            `json:"prompt_tokens,omitempty"`
	CompletionTokens   int            `json:"completion_tokens,omitempty"`
	TotalTokens        int            `json:"total_tokens,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
}

// GetMessagesRequest represents a request to get messages for a conversation
type GetMessagesRequest struct {
	NotebookID     string `json:"notebook_id"`
	ConversationID string `json:"conversation_id"` // can be empty to get latest
	Limit          int    `json:"limit"`
	BeforeSequence *int   `json:"before_sequence,omitempty"`
}

// GetMessagesResponse represents a paginated list of messages
type GetMessagesResponse struct {
	Messages       []MessageResponse `json:"messages"`
	ConversationID string            `json:"conversation_id"`
	HasMore        bool              `json:"has_more"`
	OldestSequence *int              `json:"oldest_sequence,omitempty"`
}

// ToMessageResponse maps a message entity to a response DTO
func ToMessageResponse(message *entities.Message) *MessageResponse {
	if message == nil {
		return nil
	}

	var msgData map[string]any
	if message.Message != nil {
		msgData = map[string]any{
			"role":    message.Message.Role,
			"content": message.Message.Content,
		}
	}

	return &MessageResponse{
		ID:                 message.ID,
		ConversationID:     message.ConversationID,
		SequenceNum:        message.SequenceNum,
		ResponseID:         message.ResponseID,
		PreviousResponseID: message.PreviousResponseID,
		Message:            msgData,
		Model:              message.Model,
		FinishReason:       message.FinishReason,
		PromptTokens:       message.PromptTokens,
		CompletionTokens:   message.CompletionTokens,
		TotalTokens:        message.TotalTokens,
		CreatedAt:          message.CreatedAt,
	}
}

// ToMessageResponses maps a slice of message entities to response DTOs
func ToMessageResponses(messages []*entities.Message) []MessageResponse {
	responses := make([]MessageResponse, 0, len(messages))
	for _, message := range messages {
		if message != nil {
			if resp := ToMessageResponse(message); resp != nil {
				responses = append(responses, *resp)
			}
		}
	}
	return responses
}
