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
	msg3 := &schema.Message{Content: "!"}
	
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
	}

	assert.Equal(t, len(expectedEvents), len(lines))

	for i, line := range lines {
		assert.True(t, strings.HasPrefix(line, "data: "))
		data := strings.TrimPrefix(line, "data: ")
		
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
				Output []struct {
					Type    string `json:"type"`
					Role    string `json:"role"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"output"`
			}
			var compData struct {
				Response concreteResponseResource `json:"response"`
			}

			err := json.Unmarshal([]byte(data), &compData)
			assert.NoError(t, err)
			assert.Equal(t, "completed", compData.Response.Status)
			assert.Equal(t, "Hello world!", compData.Response.Output[0].Content[0].Text)
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
