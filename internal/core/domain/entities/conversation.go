package entities

import (
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/schema"
	appuuid "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// StoredMessage represents a single message in conversation storage
// It supports both simple text content and structured multimodal content
type StoredMessage struct {
	Role      string                 `json:"role"`
	Content   interface{}            `json:"content"`              // string or StructuredContent
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// StructuredContent represents the full structured content of a message
// including tool calls, reasoning, and multimodal parts
type StructuredContent struct {
	Text                      string                  `json:"text,omitempty"`
	ReasoningContent          string                  `json:"reasoning_content,omitempty"`
	ToolCalls                 []StoredToolCall        `json:"tool_calls,omitempty"`
	AssistantGenMultiContent  []StoredMessagePart     `json:"assistant_gen_multi_content,omitempty"`
}

// StoredToolCall represents a tool call in storage
type StoredToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Arguments string `json:"arguments"`
	Index    *int   `json:"index,omitempty"`
}

// StoredMessagePart represents a multimodal content part in storage
type StoredMessagePart struct {
	Type       string                 `json:"type"`
	Text       string                 `json:"text,omitempty"`
	ImageURL   string                 `json:"image_url,omitempty"`
	AudioURL   string                 `json:"audio_url,omitempty"`
	VideoURL   string                 `json:"video_url,omitempty"`
	Reasoning  *StoredReasoning       `json:"reasoning,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// StoredReasoning represents reasoning content in storage
type StoredReasoning struct {
	Text      string `json:"text,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// ToEinoMessage converts StoredMessage to Eino's schema.Message
// This handles both simple text messages and structured multimodal messages
func (sm *StoredMessage) ToEinoMessage() *schema.Message {
	msg := &schema.Message{
		Role:      schema.RoleType(sm.Role),
		Extra:     sm.Extra,
	}

	// Handle structured content
	if structured, ok := sm.Content.(map[string]interface{}); ok {
		// Parse structured content
		if text, ok := structured["text"].(string); ok {
			msg.Content = text
		}
		if reasoning, ok := structured["reasoning_content"].(string); ok {
			msg.ReasoningContent = reasoning
		}
		if toolCallsData, ok := structured["tool_calls"].([]interface{}); ok {
			msg.ToolCalls = convertStoredToolCalls(toolCallsData)
		}
		if multiContentData, ok := structured["assistant_gen_multi_content"].([]interface{}); ok {
			msg.AssistantGenMultiContent = convertStoredMessageParts(multiContentData)
		}
	} else if str, ok := sm.Content.(string); ok {
		// Simple text content
		msg.Content = str
	}

	return msg
}

// convertStoredToolCalls converts stored tool calls back to Eino schema
func convertStoredToolCalls(data []interface{}) []schema.ToolCall {
	toolCalls := make([]schema.ToolCall, 0, len(data))
	for _, item := range data {
		tcMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tc := schema.ToolCall{}
		if id, ok := tcMap["id"].(string); ok {
			tc.ID = id
		}
		if tcType, ok := tcMap["type"].(string); ok {
			tc.Type = tcType
		}
		if name, ok := tcMap["name"].(string); ok {
			tc.Function.Name = name
		}
		if args, ok := tcMap["arguments"].(string); ok {
			tc.Function.Arguments = args
		}
		if index, ok := tcMap["index"].(float64); ok {
			idx := int(index)
			tc.Index = &idx
		}
		toolCalls = append(toolCalls, tc)
	}
	return toolCalls
}

// convertStoredMessageParts converts stored message parts back to Eino schema
func convertStoredMessageParts(data []interface{}) []schema.MessageOutputPart {
	parts := make([]schema.MessageOutputPart, 0, len(data))
	for _, item := range data {
		partMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		part := schema.MessageOutputPart{}
		if pType, ok := partMap["type"].(string); ok {
			part.Type = schema.ChatMessagePartType(pType)
		}
		if text, ok := partMap["text"].(string); ok {
			part.Text = text
		}
		if reasoning, ok := partMap["reasoning"].(map[string]interface{}); ok {
			part.Reasoning = &schema.MessageOutputReasoning{}
			if rText, ok := reasoning["text"].(string); ok {
				part.Reasoning.Text = rText
			}
			if rSig, ok := reasoning["signature"].(string); ok {
				part.Reasoning.Signature = rSig
			}
		}
		if extra, ok := partMap["extra"].(map[string]interface{}); ok {
			part.Extra = extra
		}
		parts = append(parts, part)
	}
	return parts
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

// MessageToStoredContent converts a schema.Message to structured content for storage
func MessageToStoredContent(msg *schema.Message) interface{} {
	// Check if message has structured content (tool calls, multimodal parts, reasoning)
	hasStructuredContent := len(msg.ToolCalls) > 0 ||
		len(msg.AssistantGenMultiContent) > 0 ||
		msg.ReasoningContent != ""

	if !hasStructuredContent {
		// Simple text message
		return msg.Content
	}

	// Build structured content
	structured := StructuredContent{
		Text:             msg.Content,
		ReasoningContent: msg.ReasoningContent,
	}

	// Store tool calls
	if len(msg.ToolCalls) > 0 {
		structured.ToolCalls = make([]StoredToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			structured.ToolCalls[i] = StoredToolCall{
				ID:       tc.ID,
				Type:     tc.Type,
				Name:     tc.Function.Name,
				Arguments: tc.Function.Arguments,
				Index:    tc.Index,
			}
		}
	}

	// Store multimodal content
	if len(msg.AssistantGenMultiContent) > 0 {
		structured.AssistantGenMultiContent = make([]StoredMessagePart, len(msg.AssistantGenMultiContent))
		for i, part := range msg.AssistantGenMultiContent {
			storedPart := StoredMessagePart{
				Type: string(part.Type),
				Text: part.Text,
				Extra: part.Extra,
			}

			// Handle reasoning parts
			if part.Reasoning != nil {
				storedPart.Reasoning = &StoredReasoning{
					Text:      part.Reasoning.Text,
					Signature: part.Reasoning.Signature,
				}
			}

			// Handle image/audio/video parts
			if part.Image != nil {
				if part.Image.URL != nil {
					storedPart.ImageURL = *part.Image.URL
				}
			}
			if part.Audio != nil {
				if part.Audio.URL != nil {
					storedPart.AudioURL = *part.Audio.URL
				}
			}
			if part.Video != nil {
				if part.Video.URL != nil {
					storedPart.VideoURL = *part.Video.URL
				}
			}

			structured.AssistantGenMultiContent[i] = storedPart
		}
	}

	// Convert to map for JSON storage
	result, _ := json.Marshal(structured)
	var resultMap map[string]interface{}
	json.Unmarshal(result, &resultMap)
	return resultMap
}
