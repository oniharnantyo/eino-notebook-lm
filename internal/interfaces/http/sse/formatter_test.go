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
	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
)

func TestResponsesAPIFormatter_WriteResponse(t *testing.T) {
	// Prepare mock stream
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
	formatter := NewResponsesAPIFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n\n")
	
	expectedEvents := []string{
		"response.created",
		"response.in_progress",
		"response.output_item.added",
		"response.content_part.added",
		"response.output_text.delta", // "Hello"
		"response.output_text.delta", // " world"
		"response.output_text.delta", // "!"
		"response.output_text.done",
		"response.content_part.done",
		"response.output_item.done",
		"response.completed",
		"[DONE]",
	}

	assert.Equal(t, len(expectedEvents), len(lines))

	for i, line := range lines {
		assert.True(t, strings.HasPrefix(line, "data: "))
		data := strings.TrimPrefix(line, "data: ")
		
		if data == "[DONE]" {
			assert.Equal(t, "[DONE]", expectedEvents[i])
			continue
		}

		var base struct {
			Type           string `json:"type"`
			SequenceNumber int    `json:"sequence_number"`
		}
		err := json.Unmarshal([]byte(data), &base)
		assert.NoError(t, err)
		
		assert.Equal(t, expectedEvents[i], base.Type)
		assert.Equal(t, i, base.SequenceNumber)

		// Extra checks for specific events
		if base.Type == "response.output_text.delta" {
			var deltaEvt dtos.ResponseOutputTextDeltaEvent
			err := json.Unmarshal([]byte(data), &deltaEvt)
			assert.NoError(t, err)
			if i == 4 {
				assert.Equal(t, "Hello", deltaEvt.Delta)
			} else if i == 5 {
				assert.Equal(t, " world", deltaEvt.Delta)
			} else if i == 6 {
				assert.Equal(t, "!", deltaEvt.Delta)
			}
		}

		if base.Type == "response.completed" {
			// Use a concrete struct for unmarshaling in tests
			type concreteResponseResource struct {
				Status string `json:"status"`
				Usage  *struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
					TotalTokens  int `json:"total_tokens"`
				} `json:"usage"`
				Output []json.RawMessage `json:"output"`
			}
			var compData struct {
				Response concreteResponseResource `json:"response"`
			}

			err := json.Unmarshal([]byte(data), &compData)
			assert.NoError(t, err)
			assert.Equal(t, "completed", compData.Response.Status)
			
			// Verify output has message
			assert.GreaterOrEqual(t, len(compData.Response.Output), 1)
			
			assert.NotNil(t, compData.Response.Usage)
			assert.Equal(t, 10, compData.Response.Usage.InputTokens)
			assert.Equal(t, 5, compData.Response.Usage.OutputTokens)
			assert.Equal(t, 15, compData.Response.Usage.TotalTokens)
		}
	}
}

func TestResponsesAPIFormatter_StreamError(t *testing.T) {
	// Prepare mock stream with error
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
	formatter := NewResponsesAPIFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.Error(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n\n")
	
	// Expect events up to the error, then response.failed
	expectedEvents := []string{
		"response.created",
		"response.in_progress",
		"response.output_item.added",
		"response.content_part.added",
		"response.output_text.delta", // "Hello"
		"response.output_text.delta", // " world"
		"response.failed",
	}

	assert.Equal(t, len(expectedEvents), len(lines))
	
	lastLine := lines[len(lines)-1]
	assert.True(t, strings.HasPrefix(lastLine, "data: "))
	data := strings.TrimPrefix(lastLine, "data: ")
	
	var failEvt struct {
		Type     string `json:"type"`
		Response struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"response"`
	}
	err = json.Unmarshal([]byte(data), &failEvt)
	assert.NoError(t, err)
	assert.Equal(t, "response.failed", failEvt.Type)
	assert.Contains(t, failEvt.Response.Error.Message, "stream failed")
}

func TestResponsesAPIFormatter_InProgressCompleteness(t *testing.T) {
	// Prepare mock stream
	msg := &schema.Message{Content: "Test"}
	reader, writer := schema.Pipe[*schema.Message](10)
	go func() {
		writer.Send(msg, nil)
		writer.Close()
	}()

	meta := &StreamMeta{
		ResponseID: "resp_test",
		ModelName:  "gpt-4o",
		CreatedAt:  time.Now().Unix(),
	}

	var buf bytes.Buffer
	formatter := NewResponsesAPIFormatter()
	err := formatter.WriteResponse(&buf, reader, meta)
	assert.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n\n")

	// Find response.in_progress event (should be at index 1)
	var inProgressLine string
	for _, line := range lines {
		if strings.Contains(line, "\"type\":\"response.in_progress\"") {
			inProgressLine = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	assert.NotEmpty(t, inProgressLine, "response.in_progress event not found")

	// Unmarshal into a struct that can verify all fields
	var inProgressEvt struct {
		Type           string `json:"type"`
		SequenceNumber int    `json:"sequence_number"`
		Response       struct {
			ID                string            `json:"id"`
			Object            string            `json:"object"`
			CreatedAt         int64             `json:"created_at"`
			Status            string            `json:"status"`
			Model             string            `json:"model"`
			Output            []json.RawMessage `json:"output"`
			Tools             []json.RawMessage `json:"tools"`
			ToolChoice        string            `json:"tool_choice"`
			Truncation        string            `json:"truncation"`
			ParallelToolCalls bool              `json:"parallel_tool_calls"`
			Text              *struct {
				Format *struct {
					Type string `json:"type"`
				} `json:"format"`
			} `json:"text"`
		} `json:"response"`
	}

	err = json.Unmarshal([]byte(inProgressLine), &inProgressEvt)
	assert.NoError(t, err)

	// Verify event type and sequence number
	assert.Equal(t, "response.in_progress", inProgressEvt.Type)
	assert.Equal(t, 1, inProgressEvt.SequenceNumber)

	// Verify required fields from response.created are present
	assert.Equal(t, "resp_test", inProgressEvt.Response.ID)
	assert.Equal(t, "response", inProgressEvt.Response.Object)
	assert.Equal(t, "gpt-4o", inProgressEvt.Response.Model)
	assert.Equal(t, "in_progress", inProgressEvt.Response.Status)
	assert.Equal(t, "disabled", inProgressEvt.Response.Truncation)
	assert.True(t, inProgressEvt.Response.ParallelToolCalls)

	// Verify arrays are non-nil (empty arrays, not null)
	assert.NotNil(t, inProgressEvt.Response.Output, "Output should not be nil")
	assert.NotNil(t, inProgressEvt.Response.Tools, "Tools should not be nil")
	assert.Equal(t, 0, len(inProgressEvt.Response.Output), "Output should be empty")
	assert.Equal(t, 0, len(inProgressEvt.Response.Tools), "Tools should be empty")

	// Verify ToolChoice field is present with value "auto"
	assert.Equal(t, "auto", inProgressEvt.Response.ToolChoice, "ToolChoice should be 'auto'")

	// Verify Text field is present with format {"type": "text"}
	assert.NotNil(t, inProgressEvt.Response.Text, "Text should not be nil")
	assert.NotNil(t, inProgressEvt.Response.Text.Format, "Text.Format should not be nil")
	assert.Equal(t, "text", inProgressEvt.Response.Text.Format.Type, "Text.Format.Type should be 'text'")
}
