# Semantic Chunking Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement semantic document chunking using the `elements[]` array from Kreuzberg parser, with automatic fallback to markdown splitter for documents without element data.

**Architecture:** Create a new element-based transformer that groups document elements by title semantics, storing rich metadata (chunk_title, page_start, page_end, element_count) with each chunk. The transformer auto-detects element availability and falls back to markdown splitting when needed.

**Tech Stack:** Go 1.21+, Eino framework (github.com/cloudwego/eino), Kreuzberg parser, pgvector, Clean Architecture + DDD patterns

---

## Task 1: Create ElementType Constants

**Files:**
- Create: `pkg/transformer/element/element_type.go`

**Step 1: Create ElementType type and constants**

```go
package element

// ElementType represents the type of a document element
type ElementType string

const (
	ElementTypeTitle         ElementType = "title"
	ElementTypeNarrativeText ElementType = "narrative_text"
	ElementTypeListItem      ElementType = "list_item"
	ElementTypeTable         ElementType = "table"
	ElementTypeImage         ElementType = "image"
	ElementTypePageBreak     ElementType = "page_break"
	ElementTypeHeading       ElementType = "heading"
	ElementTypeCodeBlock     ElementType = "code_block"
	ElementTypeBlockQuote    ElementType = "block_quote"
	ElementTypeHeader        ElementType = "header"
	ElementTypeFooter        ElementType = "footer"
)

// String returns the string representation of ElementType
func (e ElementType) String() string {
	return string(e)
}

// IsValid checks if the ElementType is valid
func (e ElementType) IsValid() bool {
	switch e {
	case ElementTypeTitle, ElementTypeNarrativeText, ElementTypeListItem,
		ElementTypeTable, ElementTypeImage, ElementTypePageBreak,
		ElementTypeHeading, ElementTypeCodeBlock, ElementTypeBlockQuote,
		ElementTypeHeader, ElementTypeFooter:
		return true
	}
	return false
}

// DefaultIncludedTypes returns the default element types to include in chunks
func DefaultIncludedTypes() []ElementType {
	return []ElementType{
		ElementTypeTitle,
		ElementTypeNarrativeText,
		ElementTypeListItem,
		ElementTypeHeading,
		ElementTypeTable,
	}
}
```

**Step 2: Commit**

```bash
git add pkg/transformer/element/element_type.go
git commit -m "feat(transformer): add ElementType constants for document elements"
```

---

## Task 2: Update Kreuzberg Parser to Preserve Elements

**Files:**
- Modify: `pkg/parser/kreuzberg/kreuzberg.go:114-123`

**Step 1: Add element types to KreuzbergExtractResponse struct**

Add after line 118:
```go
// KreuzbergElement represents a semantic element from the parsed document
type KreuzbergElement struct {
	ElementID string                 `json:"element_id"`
	Type      string                 `json:"element_type"` // title, narrative_text, list_item, page_break, image
	Text      string                 `json:"text"`
	Metadata  ElementMetadata        `json:"metadata"`
}

// ElementMetadata contains metadata for a document element
type ElementMetadata struct {
	PageNumber  int                    `json:"page_number"`
	Filename    string                 `json:"filename"`
	Coordinates map[string]float64     `json:"coordinates"`
	Index       int                    `json:"element_index"`
	Additional  map[string]interface{} `json:"additional"`
}
```

**Step 2: Add Elements field to KreuzbergExtractResponse**

Modify line 115-123 to:
```go
type KreuzbergExtractResponse struct {
	Content           string                   `json:"content"`
	MimeType          string                   `json:"mime_type"`
	Metadata          map[string]interface{}  `json:"metadata"`
	Elements          []KreuzbergElement      `json:"elements"`
	Tables            []interface{}           `json:"tables"`
	DetectedLanguages []string                `json:"detected_languages"`
	Chunks            interface{}             `json:"chunks"`
	Images            interface{}             `json:"images"`
}
```

**Step 3: Store elements in document metadata**

After line 228, add:
```go
// Store elements in metadata for downstream transformers
if len(result.Elements) > 0 {
	resultMeta["elements"] = result.Elements
}
```

**Step 4: Commit**

```bash
git add pkg/parser/kreuzberg/kreuzberg.go
git commit -m "feat(parser): preserve elements from Kreuzberg response for semantic chunking"
```

---

## Task 3: Create Element Transformer Package Structure

**Files:**
- Create: `pkg/transformer/element/transformer.go`

**Step 1: Create package with basic struct and constructor**

```go
package element

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
)

// Config for element-based transformer
type Config struct {
	IncludedTypes []ElementType
	MaxChunkSize  int
}

// elementTransformer implements document.Transformer
type elementTransformer struct {
	config   *Config
	fallback document.Transformer
}

// section represents a semantic section of the document
type section struct {
	title     string
	elements  []map[string]interface{}
	startPage int
	endPage   int
}
```

**Step 2: Add constructor with markdown fallback**

Add after the struct definitions:
```go
// NewElementTransformer creates a new element-based transformer with markdown fallback
func NewElementTransformer(ctx context.Context, config *Config) (document.Transformer, error) {
	if config == nil {
		config = &Config{
			IncludedTypes: DefaultIncludedTypes(),
			MaxChunkSize:  0, // Unlimited - semantic grouping only
		}
	}

	// Create markdown fallback
	fallback, err := markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
		Headers:     map[string]string{"#": "h1", "##": "h2", "###": "h3"},
		TrimHeaders: false,
		IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
			return fmt.Sprintf("%s-chunk-%d", originalID, splitIndex)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback transformer: %w", err)
	}

	return &elementTransformer{
		config:   config,
		fallback: fallback,
	}, nil
}
```

**Step 3: Commit**

```bash
git add pkg/transformer/element/transformer.go
git commit -m "feat(transformer): add element transformer package structure with fallback"
```

---

## Task 4: Implement Transform Method with Fallback Logic

**Files:**
- Modify: `pkg/transformer/element/transformer.go`

**Step 1: Add hasElements helper method**

```go
func (t *elementTransformer) hasElements(doc *schema.Document) bool {
	elements, ok := doc.MetaData["elements"]
	if !ok {
		return false
	}
	elems, ok := elements.([]interface{})
	return ok && len(elems) > 0
}
```

**Step 2: Add Transform method with fallback**

```go
// Transform splits documents based on semantic elements
// Falls back to markdown splitter if no elements present
func (t *elementTransformer) Transform(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
	var result []*schema.Document

	for _, doc := range docs {
		if !t.hasElements(doc) {
			// Fallback to markdown transformer
			fallbackDocs, err := t.fallback.Transform(ctx, []*schema.Document{doc})
			if err != nil {
				return nil, fmt.Errorf("fallback transform failed: %w", err)
			}
			result = append(result, fallbackDocs...)
			continue
		}

		// Use semantic element chunking
		chunks := t.createSemanticChunks(ctx, doc)
		result = append(result, chunks...)
	}

	return result, nil
}
```

**Step 3: Commit**

```bash
git add pkg/transformer/element/transformer.go
git commit -m "feat(transformer): add Transform method with markdown fallback"
```

---

## Task 5: Implement Semantic Chunking Logic

**Files:**
- Modify: `pkg/transformer/element/transformer.go`

**Step 1: Add isIncluded helper**

```go
func (t *elementTransformer) isIncluded(elemType string) bool {
	if len(t.config.IncludedTypes) == 0 {
		return elemType != string(ElementTypePageBreak) // Exclude page breaks by default
	}
	for _, allowed := range t.config.IncludedTypes {
		if elemType == string(allowed) {
			return true
		}
	}
	return false
}
```

**Step 2: Add sanitizeID helper**

```go
func sanitizeID(title string) string {
	if len(title) > 50 {
		return title[:50]
	}
	return title
}
```

**Step 3: Add createSemanticChunks method**

```go
func (t *elementTransformer) createSemanticChunks(ctx context.Context, doc *schema.Document) []*schema.Document {
	elements := doc.MetaData["elements"].([]interface{})
	var chunks []*schema.Document

	currentSection := &section{
		title:     "",
		elements:  make([]map[string]interface{}, 0),
		startPage: 0,
	}

	for _, elem := range elements {
		element, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}

		elemType, _ := element["element_type"].(string)

		if !t.isIncluded(elemType) {
			continue
		}

		// Track page number
		if metadata, ok := element["metadata"].(map[string]interface{}); ok {
			if page, ok := metadata["page_number"].(float64); ok {
				if currentSection.startPage == 0 {
					currentSection.startPage = int(page)
				}
				currentSection.endPage = int(page)
			}
		}

		// New section on title
		if elemType == "title" && len(currentSection.elements) > 0 {
			chunks = append(chunks, t.createChunk(doc, currentSection))
			currentSection = &section{
				title:     element["text"].(string),
				elements:  make([]map[string]interface{}, 0),
				startPage: currentSection.endPage,
			}
		} else if elemType == "title" {
			currentSection.title = element["text"].(string)
		} else {
			currentSection.elements = append(currentSection.elements, element)
		}

		// Check max chunk size
		if t.config.MaxChunkSize > 0 && len(currentSection.elements) >= t.config.MaxChunkSize {
			chunks = append(chunks, t.createChunk(doc, currentSection))
			currentSection = &section{
				title:     currentSection.title,
				elements:  make([]map[string]interface{}, 0),
				startPage: currentSection.endPage,
			}
		}
	}

	// Don't forget the last section
	if len(currentSection.elements) > 0 || currentSection.title != "" {
		chunks = append(chunks, t.createChunk(doc, currentSection))
	}

	return chunks
}
```

**Step 4: Commit**

```bash
git add pkg/transformer/element/transformer.go
git commit -m "feat(transformer): implement semantic chunking by title sections"
```

---

## Task 6: Implement Chunk Creation with Metadata

**Files:**
- Modify: `pkg/transformer/element/transformer.go`

**Step 1: Add createChunk method**

```go
func (t *elementTransformer) createChunk(doc *schema.Document, s *section) *schema.Document {
	var content string

	// Add title if present
	if s.title != "" {
		content += "# " + s.title + "\n\n"
	}

	// Add all elements
	for _, elem := range s.elements {
		if text, ok := elem["text"].(string); ok {
			content += text + "\n\n"
		}
	}

	chunkID := fmt.Sprintf("%s-chunk-%s", doc.ID, sanitizeID(s.title))

	chunkMeta := make(map[string]interface{})
	// Copy original metadata
	for k, v := range doc.MetaData {
		chunkMeta[k] = v
	}
	// Add chunk-specific metadata
	chunkMeta["chunk_title"] = s.title
	chunkMeta["page_start"] = s.startPage
	chunkMeta["page_end"] = s.endPage
	chunkMeta["element_count"] = len(s.elements)
	chunkMeta["chunk_type"] = "semantic"

	return &schema.Document{
		ID:       chunkID,
		Content:  content,
		MetaData: chunkMeta,
	}
}
```

**Step 2: Commit**

```bash
git add pkg/transformer/element/transformer.go
git commit -m "feat(transformer): add chunk creation with rich metadata"
```

---

## Task 7: Write Unit Tests for Element Transformer

**Files:**
- Create: `pkg/transformer/element/transformer_test.go`

**Step 1: Write test for semantic chunking**

```go
package element

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElementTransformer_SemanticChunking(t *testing.T) {
	ctx := context.Background()

	transformer, err := NewElementTransformer(ctx, &Config{
		IncludedTypes: []string{"title", "narrative_text"},
		MaxChunkSize:  0,
	})
	require.NoError(t, err)

	// Create document with elements
	doc := &schema.Document{
		ID:      "test-doc",
		Content: "Test content",
		MetaData: map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{
					"element_type": "title",
					"text":         "Introduction",
					"metadata": map[string]interface{}{
						"page_number": 1.0,
					},
				},
				map[string]interface{}{
					"element_type": "narrative_text",
					"text":         "This is the first paragraph.",
					"metadata": map[string]interface{}{
						"page_number": 1.0,
					},
				},
				map[string]interface{}{
					"element_type": "title",
					"text":         "Conclusion",
					"metadata": map[string]interface{}{
						"page_number": 2.0,
					},
				},
				map[string]interface{}{
					"element_type": "narrative_text",
					"text":         "This is the conclusion.",
					"metadata": map[string]interface{}{
						"page_number": 2.0,
					},
				},
			},
		},
	}

	result, err := transformer.Transform(ctx, []*schema.Document{doc})
	require.NoError(t, err)

	// Should create 2 chunks (Introduction and Conclusion)
	assert.Equal(t, 2, len(result))

	// First chunk
	assert.Contains(t, result[0].Content, "# Introduction")
	assert.Contains(t, result[0].Content, "This is the first paragraph")
	assert.Equal(t, "Introduction", result[0].MetaData["chunk_title"])
	assert.Equal(t, 1, result[0].MetaData["page_start"])
	assert.Equal(t, 1, result[0].MetaData["page_end"])

	// Second chunk
	assert.Contains(t, result[1].Content, "# Conclusion")
	assert.Contains(t, result[1].Content, "This is the conclusion")
	assert.Equal(t, "Conclusion", result[1].MetaData["chunk_title"])
	assert.Equal(t, 2, result[1].MetaData["page_start"])
	assert.Equal(t, 2, result[1].MetaData["page_end"])
}
```

**Step 2: Write test for markdown fallback**

```go
func TestElementTransformer_MarkdownFallback(t *testing.T) {
	ctx := context.Background()

	transformer, err := NewElementTransformer(ctx, nil)
	require.NoError(t, err)

	// Document without elements - should use markdown fallback
	doc := &schema.Document{
		ID:      "test-doc",
		Content: "# Introduction\n\nThis is content.\n\n## Section 2\n\nMore content.",
		MetaData: map[string]interface{}{
			"output_format": "markdown",
		},
	}

	result, err := transformer.Transform(ctx, []*schema.Document{doc})
	require.NoError(t, err)

	// Should create chunks based on markdown headers
	assert.Greater(t, len(result), 0)
	// Chunks should have markdown-style IDs
	assert.Contains(t, result[0].ID, "chunk")
}
```

**Step 3: Run tests**

```bash
go test ./pkg/transformer/element/... -v
```

Expected: All tests pass

**Step 4: Commit**

```bash
git add pkg/transformer/element/transformer_test.go
git commit -m "test(transformer): add unit tests for semantic chunking and fallback"
```

---

## Task 8: Add Transformer Configuration

**Files:**
- Modify: `internal/infrastructure/config/config.go`

**Step 1: Locate Config struct and add TransformerConfig**

Find the Config struct and add before the closing brace:
```go
type Config struct {
	// ... existing fields ...

	Transformer TransformerConfig `mapstructure:"transformer"`
}

// TransformerConfig defines document transformer settings
type TransformerConfig struct {
	Type    string                    `mapstructure:"type" validate:"oneof=markdown element"`
	Element ElementTransformerConfig   `mapstructure:"element"`
}

// ElementTransformerConfig defines element-based transformer settings
type ElementTransformerConfig struct {
	IncludedTypes []string `mapstructure:"included_types"` // Stored as strings for env config
	MaxChunkSize  int      `mapstructure:"max_chunk_size"`
}
```

**Step 2: Add element package import and helper method**

Add to imports in config.go:
```go
	"github.com/oniharnantyo/eino-notebook/pkg/transformer/element"
```

Add helper method to convert strings to ElementTypes:
```go
// GetIncludedElementTypes converts string slice to ElementType slice
func (c *ElementTransformerConfig) GetIncludedElementTypes() []element.ElementType {
	if len(c.IncludedTypes) == 0 {
		return element.DefaultIncludedTypes()
	}

	types := make([]element.ElementType, 0, len(c.IncludedTypes))
	for _, t := range c.IncludedTypes {
		et := element.ElementType(t)
		if et.IsValid() {
			types = append(types, et)
		}
	}
	return types
}
```

**Step 3: Update setDefaults function to include transformer defaults**

Find `setDefaults` function and add:
```go
func setDefaults(s *viper.Viper) {
	// ... existing defaults ...

	// Transformer defaults - use element-based as default
	s.SetDefault("transformer.type", "element")
	s.SetDefault("transformer.element.included_types", []string{"title", "narrative_text", "list_item"})
	s.SetDefault("transformer.element.max_chunk_size", 0)
}
```

**Step 3: Update Validate method**

Find the Validate method and add transformer validation:
```go
func (c *Config) Validate() error {
	// ... existing validations ...

	if c.Transformer.Type == "" {
		return fmt.Errorf("transformer.type is required")
	}

	return nil
}
```

**Step 4: Run validation**

```bash
go build ./cmd/...
```

Expected: No errors

**Step 5: Commit**

```bash
git add internal/infrastructure/config/config.go
git commit -m "feat(config): add transformer configuration with element-based default"
```

---

## Task 9: Update serve.go to Use Configurable Transformer

**Files:**
- Modify: `cmd/serve.go:187-204`

**Step 1: Add element transformer import**

Add to imports section (around line 37):
```go
	"github.com/oniharnantyo/eino-notebook/pkg/transformer/element"
```

**Step 2: Replace markdown transformer initialization**

Replace lines 187-204 with:
```go
	// Create document transformer based on config
	var docTransformer document.Transformer
	switch cfg.Transformer.Type {
	case "markdown":
		docTransformer, err = markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
			Headers: map[string]string{
				"#":   "h1",
				"##":  "h2",
				"###": "h3",
			},
			TrimHeaders: false,
			IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
				return fmt.Sprintf("%s-chunk-%d", originalID, splitIndex)
			},
		})
		if err != nil {
			log.Warn("failed to create markdown transformer", "error", err)
			docTransformer = nil
		} else {
			log.Info("initialized", "transformer", "markdown-header-splitter")
		}

	case "element":
		docTransformer, err = element.NewElementTransformer(ctx, &element.Config{
			IncludedTypes: cfg.Transformer.Element.GetIncludedElementTypes(),
			MaxChunkSize:  cfg.Transformer.Element.MaxChunkSize,
		})
		if err != nil {
			log.Warn("failed to create element transformer", "error", err)
			// Fallback to markdown
			docTransformer, err = markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
				Headers:     map[string]string{"#": "h1", "##": "h2", "###": "h3"},
				TrimHeaders: false,
				IDGenerator: func(ctx context.Context, originalID string, splitIndex int) string {
					return fmt.Sprintf("%s-chunk-%d", originalID, splitIndex)
				},
			})
			if err != nil {
				log.Warn("failed to create fallback markdown transformer", "error", err)
				docTransformer = nil
			}
		} else {
			log.Info("initialized", "transformer", "element-based",
				"included_types", cfg.Transformer.Element.IncludedTypes,
				"max_chunk_size", cfg.Transformer.Element.MaxChunkSize)
		}

	default:
		log.Warn("unknown transformer type, defaulting to element-based", "type", cfg.Transformer.Type)
		docTransformer, err = element.NewElementTransformer(ctx, &element.Config{
			IncludedTypes: cfg.Transformer.Element.GetIncludedElementTypes(),
			MaxChunkSize:  cfg.Transformer.Element.MaxChunkSize,
		})
		if err != nil {
			log.Warn("failed to create element transformer", "error", err)
			docTransformer = nil
		} else {
			log.Info("initialized", "transformer", "element-based (default)")
		}
	}
```

**Step 3: Build and verify**

```bash
go build ./cmd/serve
```

Expected: No errors

**Step 4: Commit**

```bash
git add cmd/serve.go
git commit -m "feat(serve): use configurable transformer with element-based default"
```

---

## Task 10: Update Environment Documentation

**Files:**
- Modify: `.env.example`

**Step 1: Add transformer configuration**

Add to `.env.example`:
```bash
# Transformer Configuration
# Type: element (semantic chunking) or markdown (header-based)
TRANSFORMER_TYPE=element

# Element Transformer Configuration
TRANSFORMER_ELEMENT_INCLUDED_TYPES=title,narrative_text,list_item
TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE=0
```

**Step 2: Commit**

```bash
git add .env.example
git commit -m "docs(env): add transformer configuration examples"
```

---

## Task 11: Integration Test

**Step 1: Start the server**

```bash
make run
```

Expected: Server starts with element transformer

**Step 2: Upload a test document**

```bash
# Create a test notebook first
NOTEBOOK_ID=$(curl -s -X POST http://localhost:8080/api/notebooks \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Notebook"}' | jq -r '.id')

# Upload a PDF
curl -X POST "http://localhost:8080/api/notebooks/$NOTEBOOK_ID/knowledges" \
  -F "file=@test.pdf" \
  -F "async=false"
```

**Step 3: Verify chunks in database**

```bash
psql -d eino_notebook -c "SELECT id, metadata->>'chunk_title' as title, metadata->>'page_start' as start, metadata->>'page_end' as end FROM knowledges WHERE notebook_id = '$NOTEBOOK_ID' ORDER BY created_at DESC LIMIT 5;"
```

Expected: Chunks with chunk_title, page_start, page_end in metadata

**Step 4: Test fallback with plain text**

```bash
# Upload a plain markdown file (no elements)
echo "# Test\n\nThis is a test." > test.md

curl -X POST "http://localhost:8080/api/notebooks/$NOTEBOOK_ID/knowledges" \
  -F "file=@test.md" \
  -F "async=false"
```

Expected: Document processed successfully using markdown fallback

**Step 5: Commit documentation**

```bash
echo "# Semantic Chunking

## Overview
The element-based transformer chunks documents by semantic sections (under each title) using the elements[] array from Kreuzberg parser.

## Configuration
- TRANSFORMER_TYPE=element (default) or markdown
- TRANSFORMER_ELEMENT_INCLUDED_TYPES: Element types to include (default: title,narrative_text,list_item)
- TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE: Max elements per chunk (0 = unlimited)

## Chunk Metadata
Each chunk includes:
- chunk_title: Title of the section
- page_start: First page number
- page_end: Last page number
- element_count: Number of elements in chunk
- chunk_type: \"semantic\" or \"markdown\"
" >> docs/plans/2026-03-26-semantic-chunking.md

git add docs/plans/2026-03-26-semantic-chunking.md
git commit -m "docs: add semantic chunking documentation"
```

---

## Verification Summary

After completing all tasks:

1. **Unit tests pass**: `go test ./pkg/transformer/element/... -v`
2. **Server starts**: `make run`
3. **PDF upload creates semantic chunks**: Check database for chunk_title, page_start, page_end metadata
4. **Plain text fallback works**: Markdown files still process correctly
5. **Configuration is flexible**: Can switch between element and markdown via TRANSFORMER_TYPE

---

## References

- Eino Document Transformer: `github.com/cloudwego/eino/components/document`
- Kreuzberg Parser: `pkg/parser/kreuzberg/kreuzberg.go`
- Clean Architecture patterns: `CLAUDE.md`