package response

import (
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestHistorySavingReader_OnSave_Success(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)

	saveCount := 0
	var savedAccumulated *AccumulatedMessage
	onSave := func(accumulated *AccumulatedMessage) {
		saveCount++
		savedAccumulated = accumulated
	}

	reader := NewHistorySavingReader(pr, onSave)
	pipedReader := reader.Pipe()

	// Act
	go func() {
		defer pw.Close()
		_ = pw.Send(&schema.Message{Content: "Hello"}, nil)
		_ = pw.Send(&schema.Message{Content: " world"}, nil)
	}()

	var accumulated strings.Builder
	for {
		msg, err := pipedReader.Recv()
		if msg != nil {
			accumulated.WriteString(msg.Content)
		}
		if err != nil {
			break
		}
	}

	// Assert
	assert.Equal(t, "Hello world", accumulated.String())
	assert.Equal(t, 1, saveCount, "onSave should be called exactly once")
	assert.NotNil(t, savedAccumulated)
	assert.Equal(t, "Hello world", savedAccumulated.GetContentText())
}

func TestHistorySavingReader_OnSave_WithError(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)

	saveCount := 0
	var savedAccumulated *AccumulatedMessage
	onSave := func(accumulated *AccumulatedMessage) {
		saveCount++
		savedAccumulated = accumulated
	}

	reader := NewHistorySavingReader(pr, onSave)
	pipedReader := reader.Pipe()

	// Act
	expectedErr := errors.New("stream error")
	go func() {
		_ = pw.Send(&schema.Message{Content: "Partial"}, nil)
		_ = pw.Send(nil, expectedErr)
		pw.Close()
	}()

	var accumulated strings.Builder
	var lastErr error
	for {
		msg, err := pipedReader.Recv()
		if msg != nil {
			accumulated.WriteString(msg.Content)
		}
		if err != nil {
			lastErr = err
			break
		}
	}

	// Assert
	assert.Equal(t, "Partial", accumulated.String())
	assert.Equal(t, expectedErr, lastErr)
	assert.Equal(t, 1, saveCount, "onSave should be called exactly once even on error")
	assert.NotNil(t, savedAccumulated)
	assert.Equal(t, "Partial", savedAccumulated.GetContentText())
}

func TestHistorySavingReader_OnSave_ExactlyOnce(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)

	saveCount := 0
	onSave := func(accumulated *AccumulatedMessage) {
		saveCount++
	}

	reader := NewHistorySavingReader(pr, onSave)
	pipedReader := reader.Pipe()

	// Act
	go func() {
		_ = pw.Send(&schema.Message{Content: "Test"}, nil)
		pw.Close()
	}()

	// Read until EOF
	for {
		_, err := pipedReader.Recv()
		if err != nil {
			break
		}
	}

	// Close explicitly
	pipedReader.Close()

	// Assert
	assert.Equal(t, 1, saveCount, "onSave should be called exactly once")
}

func TestHistorySavingReader_ToolCallsPreserved(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)

	var savedAccumulated *AccumulatedMessage
	onSave := func(accumulated *AccumulatedMessage) {
		savedAccumulated = accumulated
	}

	reader := NewHistorySavingReader(pr, onSave)
	pipedReader := reader.Pipe()

	// Act
	go func() {
		defer pw.Close()
		_ = pw.Send(&schema.Message{
			Content: "Searching...",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "search",
						Arguments: `{"query":"test"}`,
					},
				},
			},
		}, nil)
		_ = pw.Send(&schema.Message{Content: " found results!"}, nil)
	}()

	// Read all messages
	for {
		_, err := pipedReader.Recv()
		if err != nil {
			break
		}
	}

	// Assert
	assert.NotNil(t, savedAccumulated)
	fullMsg := savedAccumulated.GetFullMessage()
	assert.Equal(t, "Searching... found results!", fullMsg.Content)
	assert.Len(t, fullMsg.ToolCalls, 1)
	assert.Equal(t, "search", fullMsg.ToolCalls[0].Function.Name)
	assert.Equal(t, `{"query":"test"}`, fullMsg.ToolCalls[0].Function.Arguments)
}

func TestHistorySavingReader_MultimodalContentPreserved(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)

	var savedAccumulated *AccumulatedMessage
	onSave := func(accumulated *AccumulatedMessage) {
		savedAccumulated = accumulated
	}

	reader := NewHistorySavingReader(pr, onSave)
	pipedReader := reader.Pipe()

	// Act
	go func() {
		defer pw.Close()
		_ = pw.Send(&schema.Message{
			AssistantGenMultiContent: []schema.MessageOutputPart{
				{
					Type: schema.ChatMessagePartTypeText,
					Text: "Here's an image:",
				},
			},
		}, nil)
		_ = pw.Send(&schema.Message{
			AssistantGenMultiContent: []schema.MessageOutputPart{
				{
					Type: schema.ChatMessagePartTypeImageURL,
					Image: &schema.MessageOutputImage{
						MessagePartCommon: schema.MessagePartCommon{
							URL: toPtr("https://example.com/image.jpg"),
						},
					},
				},
			},
		}, nil)
	}()

	// Read all messages
	for {
		_, err := pipedReader.Recv()
		if err != nil {
			break
		}
	}

	// Assert
	assert.NotNil(t, savedAccumulated)
	fullMsg := savedAccumulated.GetFullMessage()
	assert.Len(t, fullMsg.AssistantGenMultiContent, 2)
	assert.Equal(t, schema.ChatMessagePartTypeText, fullMsg.AssistantGenMultiContent[0].Type)
	assert.Equal(t, "Here's an image:", fullMsg.AssistantGenMultiContent[0].Text)
	assert.Equal(t, schema.ChatMessagePartTypeImageURL, fullMsg.AssistantGenMultiContent[1].Type)
}

func toPtr(s string) *string {
	return &s
}
