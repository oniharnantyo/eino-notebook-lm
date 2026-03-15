package extractor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/google/uuid"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
)

// FileContentExtractor extracts content from uploaded files
// Single Responsibility: Only handles file content extraction
type FileContentExtractor struct {
	kreuzberg   *kreuzberg.KreuzbergParser
	maxFileSize int64
}

// NewFileContentExtractor creates a new file content extractor
func NewFileContentExtractor(kreuzbergParser *kreuzberg.KreuzbergParser, maxFileSize int64) *FileContentExtractor {
	return &FileContentExtractor{
		kreuzberg:   kreuzbergParser,
		maxFileSize: maxFileSize,
	}
}

// Extract extracts content from a file
func (e *FileContentExtractor) Extract(ctx context.Context, source usecases.ContentSource) (string, map[string]interface{}, error) {
	if source.Reader == nil {
		return "", nil, fmt.Errorf("no reader provided for file extraction")
	}

	// Read file content
	content, err := io.ReadAll(source.Reader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Check file size
	if int64(len(content)) > e.maxFileSize {
		return "", nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", e.maxFileSize)
	}

	metadata := make(map[string]interface{})
	filename := source.Filename
	if filename == "" {
		filename = "unknown"
	}

	contentType := e.detectContentType(filename)

	// Add file metadata
	metadata["filename"] = filename
	metadata["file_size"] = len(content)
	metadata["content_type"] = contentType

	// For text-based files, return content directly
	if e.isTextFile(filename) {
		return string(content), metadata, nil
	}

	// For binary files (PDF, images, etc.), use Kreuzberg parser if available
	if e.isParseableFile(filename) {
		// Generate a knowledge_id for this file
		knowledgeID := uuid.New().String()

		docs, err := e.kreuzberg.Parse(ctx, strings.NewReader(string(content)), parser.WithExtraMeta(map[string]any{
			"filename":    filename,
			"knowledge_id": knowledgeID,
		}))
		if err != nil {
			// Log error but don't fail completely - return fallback message
			metadata["parse_error"] = err.Error()
			metadata["knowledge_id"] = knowledgeID
			return fmt.Sprintf("[File: %s, Size: %d bytes - Parse Error: %s]", filename, len(content), err.Error()), metadata, nil
		}

		if len(docs) > 0 && docs[0].Content != "" {
			// Assign auto-generated UUID to the document ID
			docID := uuid.New().String()
			docs[0].ID = docID

			// Merge metadata from parser
			for k, v := range docs[0].MetaData {
				metadata[k] = v
			}

			// Ensure knowledge_id is in metadata
			metadata["knowledge_id"] = knowledgeID
			metadata["doc_id"] = docID

			return docs[0].Content, metadata, nil
		}
	}

	// Fallback for unsupported binary files
	return fmt.Sprintf("[File: %s, Size: %d bytes]", filename, len(content)), metadata, nil
}

// isParseableFile checks if a file can be parsed by Kreuzberg
func (e *FileContentExtractor) isParseableFile(filename string) bool {
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
