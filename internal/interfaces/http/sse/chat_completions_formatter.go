package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// ChatCompletionsFormatter formats SSE events for OpenAI-compatible Chat Completions API
// Sends OpenAI-compliant format with extended fields (reasoning_content, extra, etc.)
type ChatCompletionsFormatter struct {
	includeExtendedFields bool // Include reasoning_content, extra, etc. in OpenAI format
}

// NewChatCompletionsFormatter creates a new ChatCompletionsFormatter instance with extended fields
func NewChatCompletionsFormatter() *ChatCompletionsFormatter {
	return &ChatCompletionsFormatter{
		includeExtendedFields: true, // Default to including extended fields
	}
}

// WriteResponse writes SSE events for chat completions streaming response
// Sends OpenAI-compatible format with extended fields (reasoning_content, extra, etc.)
func (f *ChatCompletionsFormatter) WriteResponse(w io.Writer, stream *schema.StreamReader[*schema.Message], meta *StreamMeta) error {
	defer stream.Close()

	// Generate unique ID for this completion
	chunkID := fmt.Sprintf("chatcmpl-%s", uuid.New().String())
	createdTimestamp := meta.CreatedAt
	modelName := meta.ModelName

	// Track state
	contentAdded := false
	var lastPromptTokens, lastCompletionTokens, lastTotalTokens int
	hasUsage := false
	finishReason := "stop"

	// Process chunks from the stream
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// Capture usage if present
		if chunk.ResponseMeta != nil && chunk.ResponseMeta.Usage != nil {
			lastPromptTokens = chunk.ResponseMeta.Usage.PromptTokens
			lastCompletionTokens = chunk.ResponseMeta.Usage.CompletionTokens
			lastTotalTokens = chunk.ResponseMeta.Usage.TotalTokens
			hasUsage = true
		}

		// Handle reasoning content
		if chunk.ReasoningContent != "" {
			if f.includeExtendedFields {
				// Send reasoning content as extended OpenAI chunk
				err = f.sendExtendedChunk(w, &ExtendedChatCompletionChunk{
					ChatCompletionChunk: ChatCompletionChunk{
						ID:      chunkID,
						Object:  "chat.completion.chunk",
						Created: createdTimestamp,
						Model:   modelName,
						Choices: []Choice{{Delta: &Delta{}, Index: 0}},
					},
					ReasoningContent: chunk.ReasoningContent,
					Extra:           chunk.Extra,
				})
				if err != nil {
					return err
				}
			}
			continue
		}

		// Handle tool calls
		if len(chunk.ToolCalls) > 0 {
			// Skip tool calls for now - would require implementing tool_calls delta format
			continue
		}

		// Handle text content
		if chunk.Content != "" {
			// First chunk with content - send the role
			if !contentAdded {
				err = f.sendChunk(w, &ChatCompletionChunk{
					ID:      chunkID,
					Object:  "chat.completion.chunk",
					Created: createdTimestamp,
					Model:   modelName,
					Choices: []Choice{{Delta: &Delta{Role: "assistant"}, Index: 0}},
				})
				if err != nil {
					return err
				}
				contentAdded = true
			}

			// Send content delta
			if f.includeExtendedFields {
				// Send with extended fields
				err = f.sendExtendedChunk(w, &ExtendedChatCompletionChunk{
					ChatCompletionChunk: ChatCompletionChunk{
						ID:      chunkID,
						Object:  "chat.completion.chunk",
						Created: createdTimestamp,
						Model:   modelName,
						Choices: []Choice{{Delta: &Delta{Content: chunk.Content}, Index: 0}},
					},
					Extra: chunk.Extra,
				})
				if err != nil {
					return err
				}
			} else {
				// Send standard OpenAI format
				err = f.sendChunk(w, &ChatCompletionChunk{
					ID:      chunkID,
					Object:  "chat.completion.chunk",
					Created: createdTimestamp,
					Model:   modelName,
					Choices: []Choice{{Delta: &Delta{Content: chunk.Content}, Index: 0}},
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// Send final chunk with finish reason and usage
	finalChunk := &ExtendedChatCompletionChunk{
		ChatCompletionChunk: ChatCompletionChunk{
			ID:      chunkID,
			Object:  "chat.completion.chunk",
			Created: createdTimestamp,
			Model:   modelName,
			Choices: []Choice{{
				Delta:        &Delta{},
				Index:        0,
				FinishReason: &finishReason,
			}},
		},
	}

	// Add usage if available
	if hasUsage {
		finalChunk.Usage = &Usage{
			PromptTokens:     lastPromptTokens,
			CompletionTokens: lastCompletionTokens,
			TotalTokens:      lastTotalTokens,
		}
	}

	err := f.sendExtendedChunk(w, finalChunk)
	if err != nil {
		return err
	}

	// Send [DONE] termination
	_, err = fmt.Fprint(w, "data: [DONE]\n\n")
	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}

	return err
}

// sendChunk sends a single chat completion chunk as SSE event
func (f *ChatCompletionsFormatter) sendChunk(w io.Writer, chunk *ChatCompletionChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	if err != nil {
		return err
	}

	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}

	return nil
}

// sendExtendedChunk sends a chat completion chunk with extended fields
func (f *ChatCompletionsFormatter) sendExtendedChunk(w io.Writer, chunk *ExtendedChatCompletionChunk) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", string(data))
	if err != nil {
		return err
	}

	if fl, ok := w.(http.Flusher); ok {
		fl.Flush()
	}

	return nil
}

// ChatCompletionChunk represents a single chunk in OpenAI chat completion format
type ChatCompletionChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// ExtendedChatCompletionChunk extends OpenAI format with additional fields
type ExtendedChatCompletionChunk struct {
	ChatCompletionChunk
	ReasoningContent string                 `json:"reasoning_content,omitempty"`
	Extra           map[string]interface{} `json:"extra,omitempty"`
}

// Choice represents a choice in the chat completion
type Choice struct {
	Delta        *Delta  `json:"delta"`
	Index        int     `json:"index"`
	FinishReason *string `json:"finish_reason,omitempty"`
}

// Delta represents the delta content in streaming responses
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
