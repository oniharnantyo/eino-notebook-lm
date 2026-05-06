package extractor

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/document"
)

// FileContentExtractor extracts content from uploaded files
// Single Responsibility: Only handles file content extraction
type FileContentExtractor struct {
	docParser   document.DocumentParser
	maxFileSize int64
}

// NewFileContentExtractor creates a new file content extractor
func NewFileContentExtractor(docParser document.DocumentParser, maxFileSize int64) *FileContentExtractor {
	return &FileContentExtractor{
		docParser:   docParser,
		maxFileSize: maxFileSize,
	}
}

// Extract extracts content from a file
func (e *FileContentExtractor) Extract(ctx context.Context, source usecases.ContentSource) (*ExtractionResult, error) {
	if source.Reader == nil {
		return nil, fmt.Errorf("no reader provided for file extraction")
	}

	content, err := e.readAndValidateContent(source.Reader)
	if err != nil {
		return nil, err
	}

	filename := e.sanitizeFilename(source.Filename)
	baseMetadata := e.buildFileMetadata(filename, len(content))

	// For text-based files, return content directly
	if e.isTextFile(filename) {
		return &ExtractionResult{
			Content:  string(content),
			Metadata: baseMetadata,
		}, nil
	}

	// For binary files, use Kreuzberg parser
	if e.isParseableFile(filename) {
		return e.parseBinaryFile(ctx, filename, content, baseMetadata)
	}

	return nil, fmt.Errorf("unsupported file type: %s", filename)
}

// parseBinaryFile parses binary files using Kreuzberg parser.
func (e *FileContentExtractor) parseBinaryFile(ctx context.Context, filename string, content []byte, baseMetadata map[string]interface{}) (*ExtractionResult, error) {
	results, err := e.docParser.ParseFull(ctx, strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no content extracted from file %s", filename)
	}

	// For now, we take the first result as the main one
	result := results[0]

	// Merge metadata
	metadata := make(map[string]any)
	for k, v := range baseMetadata {
		metadata[k] = v
	}
	for k, v := range result.Metadata {
		metadata[k] = v
	}
	metadata["filename"] = filename

	return &ExtractionResult{
		Content:           result.Content,
		Chunks:            result.Chunks,
		Images:            result.Images,
		Metadata:          metadata,
		DetectedLanguages: result.DetectedLanguages,
	}, nil
}

// isParseableFile checks if a file can be parsed by Kreuzberg
func (e *FileContentExtractor) isParseableFile(filename string) bool {
	if !strings.Contains(filename, ".") {
		return false
	}
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	parseableExts := map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".tiff": true,
		".bmp":  true,
	}
	return parseableExts[ext]
}

// detectContentType detects the content type from filename extension
func (e *FileContentExtractor) detectContentType(filename string) string {
	if !strings.Contains(filename, ".") {
		return "application/octet-stream"
	}
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	contentTypes := map[string]string{
		".pdf":      "application/pdf",
		".doc":      "application/msword",
		".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":      "text/plain",
		".md":       "text/markdown",
		".markdown": "text/markdown",
		".html":     "text/html",
		".htm":      "text/html",
		".json":     "application/json",
		".jpg":      "image/jpeg",
		".jpeg":     "image/jpeg",
		".png":      "image/png",
		".gif":      "image/gif",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}

// isTextFile checks if a file is a text-based file
func (e *FileContentExtractor) isTextFile(filename string) bool {
	if !strings.Contains(filename, ".") {
		return false
	}
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	textExts := map[string]bool{
		".txt":      true,
		".md":       true,
		".markdown": true,
		".html":     true,
		".htm":      true,
		".json":     true,
		".xml":      true,
		".csv":      true,
		".yaml":     true,
		".yml":      true,
	}
	return textExts[ext]
}

// readAndValidateContent reads file content and validates its size
func (e *FileContentExtractor) readAndValidateContent(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	if int64(len(content)) > e.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum limit %d", len(content), e.maxFileSize)
	}

	return content, nil
}

// sanitizeFilename removes path components and unsafe characters from a filename
func (e *FileContentExtractor) sanitizeFilename(filename string) string {
	return path.Base(filename)
}

// buildFileMetadata creates initial metadata for an uploaded file
func (e *FileContentExtractor) buildFileMetadata(filename string, size int) map[string]interface{} {
	return map[string]interface{}{
		"filename":     filename,
		"size":         size,
		"content_type": e.detectContentType(filename),
	}
}
