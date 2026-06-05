package entities

import (
	"time"

	appuuid "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Message represents a single message turn in a conversation
type Message struct {
	ID                 string                 `json:"id"`
	ConversationID     string                 `json:"conversation_id"`
	SequenceNum        int                    `json:"sequence_num"`
	ResponseID         string                 `json:"response_id"`
	PreviousResponseID *string                `json:"previous_response_id,omitempty"`
	Message            *StoredMessage         `json:"message"` // JSONB content representing StoredMessage
	Model              string                 `json:"model"`
	FinishReason       string                 `json:"finish_reason,omitempty"`
	PromptTokens       int                    `json:"prompt_tokens,omitempty"`
	CompletionTokens   int                    `json:"completion_tokens,omitempty"`
	TotalTokens        int                    `json:"total_tokens,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// NewMessage creates a new message entity
func NewMessage(
	conversationID string,
	sequenceNum int,
	responseID string,
	previousResponseID *string,
	message *StoredMessage,
	model string,
	finishReason string,
	promptTokens int,
	completionTokens int,
	totalTokens int,
) *Message {
	return &Message{
		ID:                 appuuid.New().String(),
		ConversationID:     conversationID,
		SequenceNum:        sequenceNum,
		ResponseID:         responseID,
		PreviousResponseID: previousResponseID,
		Message:            message,
		Model:              model,
		FinishReason:       finishReason,
		PromptTokens:       promptTokens,
		CompletionTokens:   completionTokens,
		TotalTokens:        totalTokens,
		CreatedAt:          time.Now(),
	}
}
