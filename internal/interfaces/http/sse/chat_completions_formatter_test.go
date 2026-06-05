package sse

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

// parseSSEData extracts data from an SSE chunk (data: ...)
func parseSSEData(chunk string) string {
	for _, line := range strings.Split(chunk, "\n") {
		if strings.HasPrefix(line, "data: ") {
			return strings.TrimPrefix(line, "data: ")
		}
	}
	return ""
}

func TestChatCompletionsFormatter_WriteResponse(t *testing.T) {
	msg1 := &schema.Message{Content: "Hello"}
	msg2 := &schema.Message{Content: " world"}
	msg3 := &schema.Message{
		Content: "!",
		ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
	}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg1, nil)
		writer.Send(msg2, nil)
		writer.Send(msg3, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_123",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")

	// Expected chunks:
	// 1. Role chunk (assistant)
	// 2. Content delta "Hello"
	// 3. Content delta " world"
	// 4. Content delta "!"
	// 5. Final chunk with finish_reason and usage
	// 6. [DONE]
	expectedCount := 6
	assert.Equal(t, expectedCount, len(chunks), "expected %d chunks", expectedCount)

	// Verify last chunk is [DONE]
	lastChunk := chunks[len(chunks)-1]
	assert.Equal(t, "data: [DONE]", lastChunk)

	// Verify structure of content chunks
	var contentDeltas []string
	for i, chunk := range chunks {
		data := parseSSEData(chunk)
		if data == "[DONE]" {
			continue
		}

		var ccChunk ChatCompletionChunk
		err := json.Unmarshal([]byte(data), &ccChunk)
		assert.NoError(t, err, "chunk %d should be valid JSON", i)

		// Verify basic structure
		assert.NotEmpty(t, ccChunk.ID, "chunk %d should have ID", i)
		assert.Equal(t, "chat.completion.chunk", ccChunk.Object, "chunk %d object type", i)
		assert.Greater(t, ccChunk.Created, int64(0), "chunk %d created timestamp", i)
		assert.Equal(t, "gpt-4o", ccChunk.Model, "chunk %d model name", i)
		assert.Len(t, ccChunk.Choices, 1, "chunk %d should have exactly one choice", i)

		choice := ccChunk.Choices[0]
		assert.Equal(t, 0, choice.Index, "chunk %d choice index", i)

		// Collect content deltas
		if choice.Delta != nil && choice.Delta.Content != "" {
			contentDeltas = append(contentDeltas, choice.Delta.Content)
		}

		// Verify final chunk structure
		if choice.FinishReason != nil {
			assert.Equal(t, "stop", *choice.FinishReason, "chunk %d finish reason", i)
			assert.NotNil(t, ccChunk.Usage, "final chunk should have usage")
			assert.Equal(t, 10, ccChunk.Usage.PromptTokens, "prompt tokens")
			assert.Equal(t, 5, ccChunk.Usage.CompletionTokens, "completion tokens")
			assert.Equal(t, 15, ccChunk.Usage.TotalTokens, "total tokens")
		}

		// Verify role chunk
		if choice.Delta != nil && choice.Delta.Role != "" {
			assert.Equal(t, "assistant", choice.Delta.Role, "chunk %d role", i)
			assert.Empty(t, choice.Delta.Content, "role chunk should not have content")
		}
	}

	// Verify content deltas match expected sequence
	assert.Equal(t, []string{"Hello", " world", "!"}, contentDeltas, "content deltas should match")
}

func TestChatCompletionsFormatter_StreamError(t *testing.T) {
	msg1 := &schema.Message{Content: "Hello"}
	msg2 := &schema.Message{Content: " world"}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg1, nil)
		writer.Send(msg2, nil)
		writer.Send(nil, errors.New("stream failed"))
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_123",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stream failed")
}

func TestChatCompletionsFormatter_EmptyStream(t *testing.T) {
	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_empty",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")

	// Should only have final chunk and [DONE]
	// No role chunk since no content was sent
	assert.Equal(t, 2, len(chunks), "expected 2 chunks (final + [DONE])")

	// Verify [DONE] termination
	lastChunk := chunks[len(chunks)-1]
	assert.Equal(t, "data: [DONE]", lastChunk)

	// Verify final chunk has finish_reason
	firstData := parseSSEData(chunks[0])
	var ccChunk ChatCompletionChunk
	err = json.Unmarshal([]byte(firstData), &ccChunk)
	assert.NoError(t, err)
	assert.NotNil(t, ccChunk.Choices[0].FinishReason)
	assert.Equal(t, "stop", *ccChunk.Choices[0].FinishReason)
}

func TestChatCompletionsFormatter_ReasoningContent(t *testing.T) {
	msg1 := &schema.Message{ReasoningContent: "Thinking..."}
	msg2 := &schema.Message{Content: "Answer"}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg1, nil)
		writer.Send(msg2, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_reasoning",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")

	// With extended fields, reasoning content IS sent
	// role chunk + reasoning chunk + content delta + final chunk + [DONE]
	assert.Equal(t, 5, len(chunks))

	// Verify reasoning content is properly included in extended format
	foundReasoning := false
	for _, chunk := range chunks {
		data := parseSSEData(chunk)
		if data == "[DONE]" {
			continue
		}
		// Check for reasoning_content in extended format
		if strings.Contains(data, "reasoning_content") && strings.Contains(data, "Thinking") {
			foundReasoning = true
		}
	}
	assert.True(t, foundReasoning, "reasoning content should be included in extended format")

	// Verify we still got the regular content
	var contentDeltas []string
	for _, chunk := range chunks {
		data := parseSSEData(chunk)
		if data == "[DONE]" {
			continue
		}
		var ccChunk ChatCompletionChunk
		err := json.Unmarshal([]byte(data), &ccChunk)
		assert.NoError(t, err)
		if ccChunk.Choices[0].Delta != nil && ccChunk.Choices[0].Delta.Content != "" {
			contentDeltas = append(contentDeltas, ccChunk.Choices[0].Delta.Content)
		}
	}
	assert.Equal(t, []string{"Answer"}, contentDeltas)
}

func TestChatCompletionsFormatter_ToolCalls(t *testing.T) {
	// Test message with ONLY tool calls (no content)
	msg := &schema.Message{
		ToolCalls: []schema.ToolCall{
			{
				ID: "call_123",
				Function: schema.FunctionCall{
					Name:      "search",
					Arguments: `{"query":"test"}`,
				},
			},
		},
	}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_tools",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	// Tool calls are currently skipped in ChatCompletions format
	output := buf.String()
	assert.NotContains(t, output, "search", "tool calls should not appear in output")
	assert.NotContains(t, output, "call_123", "tool call IDs should not appear in output")

	// Should only have final chunk and [DONE] since tool calls are skipped
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")
	assert.Equal(t, 2, len(chunks), "should only have final chunk and [DONE]")
}

func TestChatCompletionsFormatter_ContentAndToolCalls(t *testing.T) {
	// Test message with BOTH content and tool calls
	// Current implementation skips the entire chunk when tool calls are present
	msg := &schema.Message{
		Content: "Response",
		ToolCalls: []schema.ToolCall{
			{
				ID: "call_123",
				Function: schema.FunctionCall{
					Name:      "search",
					Arguments: `{"query":"test"}`,
				},
			},
		},
	}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_mixed",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	// Both content and tool calls are skipped when tool calls are present
	output := buf.String()
	assert.NotContains(t, output, "Response", "content is skipped when tool calls are present")
	assert.NotContains(t, output, "search", "tool calls should not appear in output")

	// Should only have final chunk and [DONE]
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")
	assert.Equal(t, 2, len(chunks), "should only have final chunk and [DONE] when tool calls present")
}

func TestChatCompletionsFormatter_UsageStatistics(t *testing.T) {
	msg := &schema.Message{
		Content: "Test content",
		ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		},
	}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_usage",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")

	// Find the final chunk with usage
	var usageChunk *ChatCompletionChunk
	for _, chunk := range chunks {
		data := parseSSEData(chunk)
		if data == "[DONE]" {
			continue
		}
		var ccChunk ChatCompletionChunk
		err := json.Unmarshal([]byte(data), &ccChunk)
		assert.NoError(t, err)
		if ccChunk.Usage != nil {
			usageChunk = &ccChunk
			break
		}
	}

	assert.NotNil(t, usageChunk, "should have a chunk with usage statistics")
	assert.Equal(t, 100, usageChunk.Usage.PromptTokens, "prompt tokens should match")
	assert.Equal(t, 50, usageChunk.Usage.CompletionTokens, "completion tokens should match")
	assert.Equal(t, 150, usageChunk.Usage.TotalTokens, "total tokens should match")
	assert.NotNil(t, usageChunk.Choices[0].FinishReason, "final chunk should have finish reason")
}

func TestChatCompletionsFormatter_SingleChunk(t *testing.T) {
	msg := &schema.Message{
		Content: "Single",
		ResponseMeta: &schema.ResponseMeta{
			Usage: &schema.TokenUsage{
				PromptTokens:     5,
				CompletionTokens: 2,
				TotalTokens:      7,
			},
		},
	}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_single",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")

	// Role chunk + content delta + final chunk + [DONE]
	assert.Equal(t, 4, len(chunks))

	// Verify content delta
	var foundContent bool
	for _, chunk := range chunks {
		data := parseSSEData(chunk)
		if data == "[DONE]" {
			continue
		}
		var ccChunk ChatCompletionChunk
		err := json.Unmarshal([]byte(data), &ccChunk)
		assert.NoError(t, err)
		if ccChunk.Choices[0].Delta != nil && ccChunk.Choices[0].Delta.Content == "Single" {
			foundContent = true
		}
	}
	assert.True(t, foundContent, "should find content delta with 'Single'")
}

func TestChatCompletionsFormatter_ChunkStructure(t *testing.T) {
	msg := &schema.Message{Content: "Test"}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_structure",
		ModelName:  "gpt-4o",
		CreatedAt:  1234567890,
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()
	chunks := strings.Split(strings.TrimRight(output, "\n"), "\n\n")

	// Verify each chunk has proper SSE format
	for i, chunk := range chunks {
		// Skip [DONE]
		if strings.Contains(chunk, "[DONE]") {
			continue
		}

		// Verify SSE format: data: <json>
		assert.True(t, strings.HasPrefix(chunk, "data:"), "chunk %d should start with 'data:'", i)

		data := parseSSEData(chunk)
		assert.NotEmpty(t, data, "chunk %d should have data", i)

		// Verify JSON structure
		var ccChunk ChatCompletionChunk
		err := json.Unmarshal([]byte(data), &ccChunk)
		assert.NoError(t, err, "chunk %d should be valid JSON", i)

		// Verify OpenAI format fields
		assert.NotEmpty(t, ccChunk.ID, "chunk %d should have ID", i)
		assert.Equal(t, "chat.completion.chunk", ccChunk.Object, "chunk %d object type", i)
		assert.Equal(t, int64(1234567890), ccChunk.Created, "chunk %d created timestamp", i)
		assert.Equal(t, "gpt-4o", ccChunk.Model, "chunk %d model name", i)

		// Verify choices structure
		assert.Len(t, ccChunk.Choices, 1, "chunk %d should have exactly one choice", i)
		assert.NotNil(t, ccChunk.Choices[0].Delta, "chunk %d should have delta", i)
		assert.Equal(t, 0, ccChunk.Choices[0].Index, "chunk %d choice index should be 0", i)
	}
}

func TestChatCompletionsFormatter_DoneTermination(t *testing.T) {
	msg := &schema.Message{Content: "Final test"}

	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_done",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewChatCompletionsFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	output := buf.String()

	// Verify [DONE] is present and properly formatted
	assert.True(t, strings.Contains(output, "data: [DONE]"), "output should contain [DONE] termination")
	assert.True(t, strings.HasSuffix(output, "data: [DONE]\n\n"), "output should end with [DONE]")

	// Verify [DONE] is a separate chunk
	chunks := strings.Split(strings.TrimSpace(output), "\n\n")
	lastChunk := chunks[len(chunks)-1]
	assert.Equal(t, "data: [DONE]", lastChunk, "last chunk should be [DONE]")
}
