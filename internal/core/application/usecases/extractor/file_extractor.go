package extractor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"
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
func (e *FileContentExtractor) Extract(ctx context.Context, source usecases.ContentSource) ([]*schema.Document, error) {
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
		return e.buildTextDocument(string(content), baseMetadata), nil
	}

	// For binary files, use Kreuzberg parser
	if e.isParseableFile(filename) {
		return e.parseBinaryFile(ctx, filename, content, baseMetadata)
	}

	return nil, fmt.Errorf("unsupported file type: %s", filename)
}

// buildTextDocument creates a single document from text content.
func (e *FileContentExtractor) buildTextDocument(content string, baseMetadata map[string]interface{}) []*schema.Document {
	return []*schema.Document{
		{
			Content:  content,
			MetaData: baseMetadata,
		},
	}
}

// readAndValidateContent reads and validates file content.
func (e *FileContentExtractor) readAndValidateContent(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	if int64(len(content)) > e.maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", e.maxFileSize)
	}

	return content, nil
}

// sanitizeFilename ensures filename is not empty.
func (e *FileContentExtractor) sanitizeFilename(filename string) string {
	if filename == "" {
		return "unknown"
	}
	return filename
}

// buildFileMetadata creates file metadata.
func (e *FileContentExtractor) buildFileMetadata(filename string, fileSize int) map[string]interface{} {
	return map[string]interface{}{
		"filename":     filename,
		"file_size":    fileSize,
		"content_type": e.detectContentType(filename),
	}
}

// parseBinaryFile parses binary files using Kreuzberg parser.
func (e *FileContentExtractor) parseBinaryFile(ctx context.Context, filename string, content []byte, baseMetadata map[string]interface{}) ([]*schema.Document, error) {
	knowledgeID := uuid.New().String()

	docs, err := e.kreuzberg.Parse(ctx, strings.NewReader(string(content)),
		parser.WithExtraMeta(map[string]any{
			"filename":     filename,
			"knowledge_id": knowledgeID,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no content extracted from file %s", filename)
	}

	return e.buildMultiPageResult(docs, knowledgeID, baseMetadata)
}

// buildMultiPageResult prepares documents and builds metadata.
func (e *FileContentExtractor) buildMultiPageResult(docs []*schema.Document, knowledgeID string, baseMetadata map[string]interface{}) ([]*schema.Document, error) {
	// Assign UUIDs to all documents
	docIDs := e.assignDocumentIDs(docs)

	// Build final metadata and merge into base metadata
	metadata := e.buildDocumentsMetadata(docs, knowledgeID, docIDs, baseMetadata)

	// Merge metadata into each document
	for _, doc := range docs {
		for k, v := range metadata {
			if _, exists := doc.MetaData[k]; !exists {
				doc.MetaData[k] = v
			}
		}
	}

	return docs, nil
}

// assignDocumentIDs assigns unique IDs to all documents.
func (e *FileContentExtractor) assignDocumentIDs(docs []*schema.Document) []string {
	docIDs := make([]string, len(docs))
	for i, doc := range docs {
		docID := uuid.New().String()
		doc.ID = docID
		docIDs[i] = docID
	}
	return docIDs
}

// concatenateDocuments concatenates all document contents with page separators.
func (e *FileContentExtractor) concatenateDocuments(docs []*schema.Document) string {
	var b strings.Builder
	for i, doc := range docs {
		if doc.Content == "" {
			continue
		}
		if i > 0 {
			b.WriteString("\n\n--- PAGE ")
			if pageNum, ok := doc.MetaData["page_number"].(int); ok {
				b.WriteString(fmt.Sprintf("%d", pageNum))
			} else {
				b.WriteString(fmt.Sprintf("%d", i+1))
			}
			b.WriteString(" ---\n\n")
		}
		b.WriteString(doc.Content)
	}
	return b.String()
}

// buildDocumentsMetadata builds metadata for multiple documents.
func (e *FileContentExtractor) buildDocumentsMetadata(docs []*schema.Document, knowledgeID string, docIDs []string, baseMetadata map[string]interface{}) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Copy base metadata
	for k, v := range baseMetadata {
		metadata[k] = v
	}

	// Add first document's metadata as base
	if len(docs) > 0 && docs[0].Content != "" {
		for k, v := range docs[0].MetaData {
			metadata[k] = v
		}
	}

	// Add multi-page metadata
	metadata["knowledge_id"] = knowledgeID
	metadata["doc_ids"] = docIDs
	metadata["page_count"] = len(docs)

	// Store individual page metadata
	pagesMeta := make([]map[string]interface{}, len(docs))
	for i, doc := range docs {
		pagesMeta[i] = doc.MetaData
	}
	metadata["pages"] = pagesMeta

	return metadata
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
