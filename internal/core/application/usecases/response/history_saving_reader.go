package response

import (
	"strings"

	"github.com/cloudwego/eino/schema"
)

// AccumulatedMessage represents the complete accumulated message with all its parts
type AccumulatedMessage struct {
	Messages []*schema.Message
}

// historySavingReader wraps a StreamReader and triggers a save callback when the stream ends.
type historySavingReader struct {
	inner  *schema.StreamReader[*schema.Message]
	onSave func(*AccumulatedMessage)
}

// NewHistorySavingReader creates a new historySavingReader.
func NewHistorySavingReader(inner *schema.StreamReader[*schema.Message], onSave func(*AccumulatedMessage)) *historySavingReader {
	return &historySavingReader{
		inner:  inner,
		onSave: onSave,
	}
}

// Pipe returns a new StreamReader that forwards messages from the inner reader
// and triggers the onSave callback when the stream is finished or closed.
func (r *historySavingReader) Pipe() *schema.StreamReader[*schema.Message] {
	sr, sw := schema.Pipe[*schema.Message](10)

	go func() {
		defer sw.Close()
		defer r.inner.Close()

		accumulated := &AccumulatedMessage{
			Messages: []*schema.Message{},
		}

		for {
			msg, err := r.inner.Recv()
			if msg != nil {
				// Accumulate all message parts (including tool calls, multimodal content, etc.)
				accumulated.Messages = append(accumulated.Messages, msg)
				_ = sw.Send(msg, nil)
			}
			if err != nil {
				r.onSave(accumulated)
				_ = sw.Send(nil, err)
				break
			}
		}
	}()

	return sr
}

// GetContentText extracts the plain text content from accumulated messages
// This is used for backward compatibility with ResponseText field
func (am *AccumulatedMessage) GetContentText() string {
	var sb strings.Builder
	for _, msg := range am.Messages {
		if msg.Content != "" {
			sb.WriteString(msg.Content)
		}
	}
	return sb.String()
}

// GetFullMessage combines all accumulated messages into a single complete message
// Accumulates consecutive chunks of the same type (reasoning, content) into single blocks
// while preserving tool calls as discrete events in chronological order
func (am *AccumulatedMessage) GetFullMessage() *schema.Message {
	if len(am.Messages) == 0 {
		return &schema.Message{}
	}

	combined := &schema.Message{
		Role:  am.Messages[0].Role,
		Extra: am.Messages[0].Extra,
	}

	// Accumulate parts by type, merging consecutive chunks
	var parts []schema.MessageOutputPart
	var reasoningBuffer strings.Builder
	var contentBuffer strings.Builder
	var hasMultimodal bool

	for _, msg := range am.Messages {
		// Handle reasoning - accumulate into buffer
		if msg.ReasoningContent != "" {
			if reasoningBuffer.Len() == 0 {
				// Start of new reasoning block
			}
			reasoningBuffer.WriteString(msg.ReasoningContent)
		}

		// Handle content - accumulate into buffer
		if msg.Content != "" {
			if contentBuffer.Len() == 0 {
				// Start of new content block
			}
			contentBuffer.WriteString(msg.Content)
		}

		// Handle tool calls - flush buffers and add tool call as discrete event
		if len(msg.ToolCalls) > 0 {
			// Flush any pending reasoning before tool call
			if reasoningBuffer.Len() > 0 {
				parts = append(parts, schema.MessageOutputPart{
					Type: schema.ChatMessagePartTypeReasoning,
					Reasoning: &schema.MessageOutputReasoning{
						Text: reasoningBuffer.String(),
					},
				})
				reasoningBuffer.Reset()
				hasMultimodal = true
			}

			// Flush any pending content before tool call
			if contentBuffer.Len() > 0 {
				parts = append(parts, schema.MessageOutputPart{
					Type: schema.ChatMessagePartTypeText,
					Text: contentBuffer.String(),
				})
				contentBuffer.Reset()
				hasMultimodal = true
			}

			// Add tool call(s) as discrete events
			for _, tc := range msg.ToolCalls {
				parts = append(parts, schema.MessageOutputPart{
					Type: "tool_call",
					Extra: map[string]interface{}{
						"id":       tc.ID,
						"type":     tc.Type,
						"name":     tc.Function.Name,
						"arguments": tc.Function.Arguments,
						"index":    tc.Index,
					},
				})
				hasMultimodal = true
			}
		}

		// Handle existing multimodal content
		if len(msg.AssistantGenMultiContent) > 0 {
			// Flush buffers before multimodal content
			if reasoningBuffer.Len() > 0 {
				parts = append(parts, schema.MessageOutputPart{
					Type: schema.ChatMessagePartTypeReasoning,
					Reasoning: &schema.MessageOutputReasoning{
						Text: reasoningBuffer.String(),
					},
				})
				reasoningBuffer.Reset()
				hasMultimodal = true
			}

			if contentBuffer.Len() > 0 {
				parts = append(parts, schema.MessageOutputPart{
					Type: schema.ChatMessagePartTypeText,
					Text: contentBuffer.String(),
				})
				contentBuffer.Reset()
				hasMultimodal = true
			}

			parts = append(parts, msg.AssistantGenMultiContent...)
			hasMultimodal = true
		}
	}

	// Flush any remaining buffers at the end
	if reasoningBuffer.Len() > 0 {
		parts = append(parts, schema.MessageOutputPart{
			Type: schema.ChatMessagePartTypeReasoning,
			Reasoning: &schema.MessageOutputReasoning{
				Text: reasoningBuffer.String(),
			},
		})
		hasMultimodal = true
	}

	if contentBuffer.Len() > 0 {
		parts = append(parts, schema.MessageOutputPart{
			Type: schema.ChatMessagePartTypeText,
			Text: contentBuffer.String(),
		})
		hasMultimodal = true
	}

	// Set multimodal content if we have parts
	if hasMultimodal {
		combined.AssistantGenMultiContent = parts
	}

	// Also set flat fields for backward compatibility
	var reasoningBuilder, contentBuilder strings.Builder
	var toolCalls []schema.ToolCall

	for _, msg := range am.Messages {
		if msg.ReasoningContent != "" {
			reasoningBuilder.WriteString(msg.ReasoningContent)
		}
		if msg.Content != "" {
			contentBuilder.WriteString(msg.Content)
		}
		if len(msg.ToolCalls) > 0 {
			toolCalls = append(toolCalls, msg.ToolCalls...)
		}
	}

	combined.ReasoningContent = reasoningBuilder.String()
	combined.Content = contentBuilder.String()
	if len(toolCalls) > 0 {
		combined.ToolCalls = toolCalls
	}

	return combined
}
