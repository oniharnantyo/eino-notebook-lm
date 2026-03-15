package extractor

import (
	"context"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
)

// TextContentExtractor extracts content from direct text input
// Single Responsibility: Only handles direct text content
type TextContentExtractor struct {
	maxLength int
}

// NewTextContentExtractor creates a new text content extractor
func NewTextContentExtractor(maxLength int) *TextContentExtractor {
	return &TextContentExtractor{
		maxLength: maxLength,
	}
}

// Extract returns the text content as-is
func (e *TextContentExtractor) Extract(ctx context.Context, source usecases.ContentSource) (string, map[string]interface{}, error) {
	if source.Text == "" {
		return "", nil, fmt.Errorf("no text provided for text extraction")
	}

	// Check length
	if len(source.Text) > e.maxLength {
		return "", nil, fmt.Errorf("text length exceeds maximum allowed length of %d characters", e.maxLength)
	}

	// Create metadata
	metadata := make(map[string]interface{})
	metadata["content_length"] = len(source.Text)
	metadata["content_type"] = "text/plain"

	// Merge with any existing metadata
	if source.Metadata != nil {
		for k, v := range source.Metadata {
			metadata[k] = v
		}
	}

	return source.Text, metadata, nil
}
