# Knowledge Creation API Documentation

## Overview

This document describes how to create knowledge entries in the eino-notebook application. Knowledge creation involves ingesting content from various sources (files, URLs, or direct text), processing it, and indexing it for vector search.

`★ Insight ─────────────────────────────────────`
**Knowledge vs Sources**: The application separates concerns - `Source` represents the raw content input (file, URL, text), while `Knowledge` represents the processed and indexed chunks ready for semantic search. When you create knowledge, you're essentially telling the system to take a Source's content and make it searchable.
`─────────────────────────────────────────────────`

## Endpoint

```
POST /api/v1/notebooks/{notebookId}/knowledge
```

## Request Format

The endpoint accepts `multipart/form-data` to support file uploads, URLs, and direct text content in a single API.

### Form Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | File | No* | Uploaded file (PDF, DOCX, Markdown, TXT, HTML) |
| `url` | string | No* | URL to fetch content from |
| `content` | string | No* | Direct text/markdown content |
| `title` | string | No | Title for the knowledge (defaults to filename or URL) |
| `source_type` | string | No | Type: `document`, `website`, `text`, `api`, `other` |
| `metadata` | JSON string | No | Additional metadata as JSON object |
| `sub_indexes` | JSON array | No | Array of sub-index names for categorization |
| `async` | boolean | No | Enable async processing (default: `false`) |

*At least one of `file`, `url`, or `content` must be provided.

### Supported File Types

| Extension | MIME Type | Description |
|-----------|-----------|-------------|
| `.pdf` | `application/pdf` | PDF documents |
| `.docx` | `application/vnd.openxmlformats-officedocument.wordprocessingml.document` | Word documents |
| `.md` | `text/markdown` | Markdown files |
| `.txt` | `text/plain` | Plain text |
| `.html`, `.htm` | `text/html` | HTML documents |

## Response Format

### Sync Response (201 Created)

```json
{
  "source_id": "uuid-here",
  "status": "completed",
  "error": null,
  "updated_at": "2026-03-27T10:00:00Z",
  "is_async": false
}
```

### Async Response (202 Accepted)

```json
{
  "source_id": "uuid-here",
  "status": "pending",
  "error": null,
  "updated_at": "2026-03-27T10:00:00Z",
  "is_async": true,
  "status_url": "/api/v1/sources/{sourceId}/status",
  "status_stream_url": "/api/v1/sources/{sourceId}/status/stream"
}
```

### Error Response

```json
{
  "code": "Bad Request",
  "message": "either file, url, or content must be provided"
}
```

## Usage Examples

### 1. Upload a PDF File

```bash
curl -X POST \
  http://localhost:8080/api/v1/notebooks/{notebookId}/knowledge \
  -F "file=@/path/to/document.pdf" \
  -F "title=My Document" \
  -F "source_type=document" \
  -F "metadata={\"author\":\"John Doe\",\"year\":2024}"
```

### 2. Ingest a Website URL

```bash
curl -X POST \
  http://localhost:8080/api/v1/notebooks/{notebookId}/knowledge \
  -F "url=https://example.com/article" \
  -F "title=Interesting Article" \
  -F "source_type=website"
```

### 3. Submit Direct Text Content

```bash
curl -X POST \
  http://localhost:8080/api/v1/notebooks/{notebookId}/knowledge \
  -F "content=This is the full text content to be indexed for search." \
  -F "title=My Notes" \
  -F "source_type=text"
```

### 4. Async Processing (Recommended for Large Files)

```bash
curl -X POST \
  http://localhost:8080/api/v1/notebooks/{notebookId}/knowledge \
  -F "file=@/path/to/large-document.pdf" \
  -F "async=true"
```

### 5. With Sub-Indexes for Categorization

```bash
curl -X POST \
  http://localhost:8080/api/v1/notebooks/{notebookId}/knowledge \
  -F "content=Machine learning is a subset of artificial intelligence..." \
  -F "title=ML Introduction" \
  -F "sub_indexes=[\"technology\",\"ai\",\"ml\"]"
```

## Async Processing Flow

When `async=true`, the request returns immediately with a `202 Accepted` status. Use the provided URLs to track progress:

### Check Status (Polling)

```bash
curl http://localhost:8080/api/v1/sources/{sourceId}/status
```

Response:
```json
{
  "source_id": "uuid-here",
  "status": "processing",  // pending, processing, completed, failed
  "error": null,
  "updated_at": "2026-03-27T10:00:00Z"
}
```

### Stream Status (Server-Sent Events)

```bash
curl -N http://localhost:8080/api/v1/sources/{sourceId}/status/stream
```

The stream will send events whenever the status changes and close when terminal (`completed` or `failed`) is reached.

`★ Insight ─────────────────────────────────────`
**Async Processing Benefits**: Large files (PDFs, DOCX) require parsing, chunking, and embedding generation which can take seconds to minutes. Async processing prevents HTTP timeouts and allows you to handle multiple ingestion requests concurrently. The SSE streaming endpoint provides real-time updates without repeated polling.
`─────────────────────────────────────────────────`

## Status Values

| Status | Description |
|--------|-------------|
| `pending` | Request queued, not yet started |
| `processing` | Content is being parsed, chunked, and embedded |
| `completed` | Successfully indexed and ready for search |
| `failed` | Processing failed (check `error` field for details) |

## Processing Pipeline

When knowledge is created, the following steps occur:

1. **Source Creation**: A `Source` entity is created to track the content
2. **Content Extraction**: Text is extracted from file/URL/content using the Extractor component
3. **Content Sanitization**: Null bytes are removed for PostgreSQL compatibility
4. **Metadata Merging**: Extracted metadata is merged with user-provided metadata
5. **Source Update**: The source is updated with extracted content and metadata
6. **Knowledge Creation**: A knowledge entry is created from the processed content
7. **Chunking**: Content is split into semantic chunks
8. **Embedding**: Each chunk is converted to a vector using the configured embedder
9. **Indexing**: Vectors are stored in pgvector for similarity search

## Extractor Component

The extractor is responsible for extracting text content from various sources. It follows the **Strategy Pattern** with separate extractors for each content type.

### Extractor Factory

The `ContentExtractorFactory` routes requests to the appropriate extractor based on `ContentType`:

| ContentType | Extractor | Description |
|-------------|-----------|-------------|
| `file` | `FileContentExtractor` | Handles uploaded files |
| `url` | `URLContentExtractor` | Fetches content from URLs |
| `text` | `TextContentExtractor` | Handles direct text input |

### File Content Extractor

Handles file uploads with the following logic:

1. **Validation**: Checks file size against `maxFileSize` limit
2. **Text Files**: For `.txt`, `.md`, `.html`, `.json`, etc., content is returned directly
3. **Binary Files**: For `.pdf`, `.docx`, `.jpg`, `.png`, etc., the Kreuzberg parser is used

**Supported Binary Formats:**
- PDF (`.pdf`)
- Word Documents (`.doc`, `.docx`)
- Images (`.jpg`, `.jpeg`, `.png`, `.gif`, `.tiff`, `.bmp`)

**Extracted Metadata:**
```json
{
  "filename": "document.pdf",
  "file_size": 12345,
  "content_type": "application/pdf",
  "page_count": 10,
  "doc_ids": ["uuid1", "uuid2", ...],
  "pages": [...]
}
```

### URL Content Extractor

Fetches content from HTTP/HTTPS URLs:

1. **Validation**: Validates URL format and scheme (http/https only)
2. **Request**: Sends GET request with user agent header
3. **Content Type Check**: Only accepts text-based content types
4. **Response**: Returns HTML/text content as a document

**Supported Content Types:**
- `text/html`, `text/plain`, `text/xml`, `text/markdown`
- `application/json`, `application/xml`, `application/xhtml+xml`

**Extracted Metadata:**
```json
{
  "url": "https://example.com/article",
  "status_code": 200,
  "content_type": "text/html",
  "content_length": 12345
}
```

### Text Content Extractor

Handles direct text input:

1. **Validation**: Checks text length against `maxLength` limit
2. **Metadata**: Adds content length and type to metadata
3. **Return**: Returns text as a single document

**Extracted Metadata:**
```json
{
  "content_length": 1234,
  "content_type": "text/plain"
}
```

## Kreuzberg Parser

The `KreuzbergParser` is an HTTP client that connects to a Kreuzberg service for advanced document parsing. It handles:

- **PDF Documents**: Text extraction with optional OCR
- **Office Documents**: Word, PowerPoint, Excel parsing
- **Images**: OCR for scanned documents and images

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ServiceURL` | string | `localhost:8000` | Kreuzberg service URL |
| `OutputFormat` | string | `plain` | Output format: `plain`, `markdown`, `djot`, `html` |
| `ToPages` | bool | `false` | Return each page as separate document |
| `Timeout` | duration | `30s` | HTTP request timeout |

### Advanced Extraction Config

```go
type ExtractConfig struct {
    EnableQualityProcessing bool      // Enable quality processing
    ForceOCR               bool      // Force OCR for all documents
    ResultFormat           string    // "element_based" for structured output
    OutputFormat           string    // plain, markdown, djot, html
    IncludeDocumentStructure bool    // Include structural elements
    OCR                    *OCRConfig // OCR configuration
    PDFOptions             *PDFOptions // PDF-specific options
}
```

### OCR Configuration

```go
type OCRConfig struct {
    Backend   string           // "paddleocr", "tesseract"
    Language  string           // "eng", "deu", "fra", etc.
    PaddleOCRConfig *PaddleOCRConfig // PaddleOCR-specific settings
}
```

### Processing Modes

1. **Plain Text Mode**: Returns all content as a single text string
2. **Pages Mode** (`ToPages: true`): Returns each page as a separate document with page separators
3. **Elements Mode** (`WithToElements: true)`): Returns structured elements grouped by page

`★ Insight ─────────────────────────────────────`
**Extractor Design**: The extractor uses the Strategy Pattern for clean separation of concerns. Each extractor handles one content type, making it easy to add new sources (e.g., database queries, API calls) without modifying existing code. The Kreuzberg parser provides advanced document parsing capabilities that would be complex to implement in-process, delegating to a specialized service instead.
`─────────────────────────────────────────────────`

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `either file, url, or content must be provided` | No content source specified | Provide at least one content source |
| `invalid notebook_id format` | Invalid UUID format | Check the notebookId in URL path |
| `Failed to parse multipart form` | Request exceeds size limit | Max upload size is 100MB |
| `Invalid metadata JSON` | Malformed metadata string | Ensure valid JSON format |

### Example Error Response

```json
{
  "code": "Bad Request",
  "message": "Invalid metadata JSON: unexpected token at position 15"
}
```

## Best Practices

1. **Use Async for Large Files**: Always set `async=true` for files larger than 10MB
2. **Provide Meaningful Titles**: Helps with knowledge retrieval and organization
3. **Use Sub-Indexes**: Categorize knowledge for better filtering during search
4. **Include Metadata**: Add contextual information (author, tags, dates) for richer search
5. **Handle Errors Gracefully**: Always check the `error` field in async status responses
