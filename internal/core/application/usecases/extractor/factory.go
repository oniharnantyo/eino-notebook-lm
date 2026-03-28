package extractor

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
)

// ContentExtractor defines the interface for extracting content from different sources
// Single Responsibility Principle: Each extractor handles one type of content
// Interface Segregation Principle: Small, focused interface
// Dependency Inversion Principle: High-level modules depend on this abstraction
type ContentExtractor interface {
	// Extract extracts content from the source and returns documents
	// Returns schema.Document slices with metadata embedded in each document
	Extract(ctx context.Context, source usecases.ContentSource) ([]*schema.Document, error)
}

// ContentExtractorFactory creates content extractors based on content type
// Factory Pattern for creating appropriate extractors
type ContentExtractorFactory interface {
	GetExtractor(contentType usecases.ContentType) (ContentExtractor, error)
}

// DefaultContentExtractorFactory is the default implementation of ContentExtractorFactory
type DefaultContentExtractorFactory struct {
	fileExtractor ContentExtractor
	urlExtractor  ContentExtractor
	textExtractor ContentExtractor
}

// NewContentExtractorFactory creates a new content extractor factory
// Dependency Injection: Extractors are injected
func NewContentExtractorFactory(
	fileExtractor ContentExtractor,
	urlExtractor ContentExtractor,
	textExtractor ContentExtractor,
) ContentExtractorFactory {
	return &DefaultContentExtractorFactory{
		fileExtractor: fileExtractor,
		urlExtractor:  urlExtractor,
		textExtractor: textExtractor,
	}
}

// GetExtractor returns the appropriate content extractor for the given content type
func (f *DefaultContentExtractorFactory) GetExtractor(contentType usecases.ContentType) (ContentExtractor, error) {
	switch contentType {
	case usecases.ContentTypeFile:
		return f.fileExtractor, nil
	case usecases.ContentTypeURL:
		return f.urlExtractor, nil
	case usecases.ContentTypeText:
		return f.textExtractor, nil
	default:
		// Return text extractor as default for unknown types
		return f.textExtractor, nil
	}
}
