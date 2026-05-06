package entities

import (
	"time"

	"github.com/cloudwego/eino/schema"
	appuuid "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// StoredMessage represents a single message in conversation storage
type StoredMessage struct {
	Role      string                 `json:"role"`
	Content   interface{}            `json:"content"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// ToEinoMessage converts StoredMessage to Eino's schema.Message
func (sm *StoredMessage) ToEinoMessage() *schema.Message {
	content := ""
	if str, ok := sm.Content.(string); ok {
		content = str
	}

	return &schema.Message{
		Role:    schema.RoleType(sm.Role),
		Content: content,
		Extra:   sm.Extra,
	}
}

// Conversation represents a stored conversation (one response)
type Conversation struct {
	ID                 string            `json:"id"`
	NotebookID         *string           `json:"notebook_id,omitempty"` // Optional association with a notebook
	PreviousResponseID *string           `json:"previous_response_id,omitempty"`
	ResponseID         string            `json:"response_id"`
	Messages           []*StoredMessage  `json:"messages"` // Full conversation history up to this point
	RequestInput       interface{}       `json:"request_input"`
	ResponseText       string            `json:"response_text"`    // Plain text for quick access
	ResponseMessage    interface{}       `json:"response_message"` // Full schema.Message as JSONB
	Model              string            `json:"model"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CreatedAt          int64             `json:"created_at"`
}

// NewConversation creates a new conversation entry
func NewConversation(
	notebookID *string,
	previousResponseID *string,
	responseID string,
	messages []*StoredMessage,
	requestInput interface{},
	responseText string,
	responseMessage interface{},
	model string,
	metadata map[string]string,
) *Conversation {
	return &Conversation{
		ID:                 appuuid.New().String(),
		NotebookID:         notebookID,
		PreviousResponseID: previousResponseID,
		ResponseID:         responseID,
		Messages:           messages,
		RequestInput:       requestInput,
		ResponseText:       responseText,
		ResponseMessage:    responseMessage,
		Model:              model,
		Metadata:           metadata,
		CreatedAt:          time.Now().Unix(),
	}
}

// GetEinoMessages converts stored messages to Eino schema.Messages
func (c *Conversation) GetEinoMessages() []*schema.Message {
	messages := make([]*schema.Message, len(c.Messages))
	for i, msg := range c.Messages {
		messages[i] = msg.ToEinoMessage()
	}
	return messages
}

// GetMessageHistoryForResponse returns messages up to and including this response
// This is used when this conversation is referenced as PreviousResponseID
func (c *Conversation) GetMessageHistoryForResponse() []*schema.Message {
	return c.GetEinoMessages()
}
