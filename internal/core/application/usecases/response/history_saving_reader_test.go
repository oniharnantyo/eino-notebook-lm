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
	var savedText string
	onSave := func(text string) {
		saveCount++
		savedText = text
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
	assert.Equal(t, "Hello world", savedText)
}

func TestHistorySavingReader_OnSave_WithError(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)
	
	saveCount := 0
	var savedText string
	onSave := func(text string) {
		saveCount++
		savedText = text
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
	assert.Equal(t, "Partial", savedText)
}

func TestHistorySavingReader_OnSave_ExactlyOnce(t *testing.T) {
	// Arrange
	pr, pw := schema.Pipe[*schema.Message](10)
	
	saveCount := 0
	onSave := func(text string) {
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
