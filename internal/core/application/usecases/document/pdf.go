package document

import (
	"context"
	"io"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
)

// PDFExtractor defines the interface for extracting text from PDF files
// Single Responsibility: Only handles PDF text extraction
// Dependency Inversion: High-level modules depend on this abstraction
type PDFExtractor interface {
	// ExtractText extracts text content from a PDF
	ExtractText(ctx context.Context, reader io.Reader, filename string) (string, map[string]interface{}, error)

	// IsAvailable checks if the extractor is available (e.g., service is running)
	IsAvailable(ctx context.Context) bool
}

// PDFExtractorConfig holds configuration for PDF extractors
type PDFExtractorConfig struct {
	// Request timeout
	Timeout int // seconds
}

// PDFExtractorFactory creates PDF extractors based on availability and configuration
// Factory Pattern + Strategy Pattern
type PDFExtractorFactory struct {
	// TODO: Add Kreuzberg or other extractors here
}

// NewPDFExtractorFactory creates a new PDF extractor factory
func NewPDFExtractorFactory(config *PDFExtractorConfig) *PDFExtractorFactory {
	return &PDFExtractorFactory{}
}

// GetExtractor returns the first available PDF extractor in the fallback chain
// Open/Closed Principle: Easy to add new extractors without modifying this method
func (f *PDFExtractorFactory) GetExtractor(ctx context.Context) PDFExtractor {
	// TODO: Add extractors here (e.g., Kreuzberg)
	return nil
}

// ExtractText extracts text from PDF using the best available extractor
func (f *PDFExtractorFactory) ExtractText(ctx context.Context, reader io.Reader, filename string) (string, map[string]interface{}, error) {
	extractor := f.GetExtractor(ctx)
	if extractor == nil {
		return "", nil, usecases.ErrNoAvailablePDFExtractor
	}
	return extractor.ExtractText(ctx, reader, filename)
}
