# Multimodal Document Ingestion Design

**Date:** 2026-03-20
**Status:** Draft
**Author:** Design Session

## Overview

This document describes the design for ingesting and storing multimodal content (tables, images) from kreuzberg document parsing output for the eino-notebook application.

## Goals

1. **Multimodal RAG** - Enable visual similarity search (e.g., "show me charts about revenue")
2. **Rich Source Preview** - Display parsed documents with tables, images, etc. in their original context
3. **Precise Text Search** - Maintain chunk-level text search precision

## Design Decisions Summary

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Image Storage | S3/Object Storage | Keeps DB lean, handles large files well |
| Table Representation | Hybrid (markdown + raw data) | Searchable via text + preserves structure for preview |
| Visual Search | Page-level embeddings (ColPali-style) | Preserves layout context, matches NotebookLM behavior |
| Text Search | Chunks with page references | Precise search, pages for display |

---

## Data Model

### Entity Relationship Diagram

```
┌─────────────────────┐
│       Source        │
│   (existing)        │
├─────────────────────┤         ┌─────────────────────┐
│ - id                │────────<│      Knowledge      │
│ - notebook_id       │         │   (existing)        │
│ - title             │         ├─────────────────────┤
│ - uri               │         │ - knowledge_id      │
│ - content_type      │         │ - source_id (FK)    │
│ - content           │         │ - title             │
│ - metadata          │         │ - content           │
│ - chunk_count       │         │ - source_type       │
└─────────────────────┘         │ - page_number (NEW) │
         │                      │ - metadata          │
         │                      └─────────────────────┘
         │
         │ 1:N
         ▼
┌─────────────────────┐         ┌─────────────────────┐
│       Page          │         │    SourceAsset      │
│    (new entity)     │         │    (new entity)     │
├─────────────────────┤         ├─────────────────────┤
│ - id                │────────<│ - id                │
│ - source_id (FK)    │         │ - source_id (FK)    │
│ - page_number       │         │ - page_id (FK)      │
│ - content           │         │ - asset_type        │
│ - embedding         │         │ - storage_url (S3)  │
│ - metadata          │         │ - bbox (position)   │
│   - width           │         │ - width             │
│   - height          │         │ - height            │
│   - is_blank        │         │ - format            │
└─────────────────────┘         │ - metadata          │
         │                      └─────────────────────┘
         │
         │ Stores raw table data
         ▼
┌─────────────────────┐
│   Source.metadata   │
│   tables: [         │
│     {               │
│       page_number,  │
│       headers[],    │
│       rows[][],     │
│       markdown      │
│     }               │
│   ]                 │
└─────────────────────┘
```

### New Entities

#### Page Entity

Stores per-page content and embeddings for visual similarity search.

```go
// internal/core/domain/entities/page.go
type Page struct {
    ID          uuid.UUID              `json:"id" db:"id"`
    SourceID    uuid.UUID              `json:"source_id" db:"source_id"`
    PageNumber  int                    `json:"page_number" db:"page_number"`
    Content     string                 `json:"content" db:"content"`           // Page text content
    Embedding   []float64              `json:"embedding,omitempty" db:"embedding"` // ColPali-style page embedding
    Metadata    map[string]any         `json:"metadata" db:"metadata"`         // width, height, is_blank, block_count
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}
```

**Database Schema:**
```sql
CREATE TABLE pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    page_number INT NOT NULL,
    content TEXT,
    embedding vector(1024),  -- ColPali embedding dimension
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id, page_number)
);

CREATE INDEX idx_pages_embedding ON pages USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_pages_source ON pages(source_id);
```

#### SourceAsset Entity

Stores references to images and other binary assets stored in S3.

```go
// internal/core/domain/entities/source_asset.go
type AssetType string

const (
    AssetTypeImage AssetType = "image"
    AssetTypeTable AssetType = "table"  // For table screenshots if needed
)

type SourceAsset struct {
    ID          uuid.UUID              `json:"id" db:"id"`
    SourceID    uuid.UUID              `json:"source_id" db:"source_id"`
    PageID      *uuid.UUID             `json:"page_id,omitempty" db:"page_id"`  // Optional link to page
    AssetType   AssetType              `json:"asset_type" db:"asset_type"`
    StorageURL  string                 `json:"storage_url" db:"storage_url"`    // S3 URL
    BBox        []float64              `json:"bbox,omitempty" db:"bbox"`        // [x1, y1, x2, y2] position on page
    Width       int                    `json:"width,omitempty" db:"width"`
    Height      int                    `json:"height,omitempty" db:"height"`
    Format      string                 `json:"format,omitempty" db:"format"`    // png, jpeg, etc.
    Metadata    map[string]any         `json:"metadata,omitempty" db:"metadata"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}
```

**Database Schema:**
```sql
CREATE TABLE source_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    page_id UUID REFERENCES pages(id) ON DELETE SET NULL,
    asset_type VARCHAR(20) NOT NULL,
    storage_url TEXT NOT NULL,
    bbox FLOAT[],  -- [x1, y1, x2, y2]
    width INT,
    height INT,
    format VARCHAR(20),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_assets_source ON source_assets(source_id);
CREATE INDEX idx_assets_page ON source_assets(page_id);
```

### Modified Entities

#### Knowledge Entity (Add Page Reference)

```go
// Add to existing Knowledge entity
type Knowledge struct {
    // ... existing fields ...
    PageNumber  *int                   `json:"page_number,omitempty" db:"page_number"`  // NEW: Reference to source page
}
```

**Migration:**
```sql
ALTER TABLE knowledge ADD COLUMN page_number INT;
CREATE INDEX idx_knowledge_page ON knowledge(source_id, page_number);
```

#### Source Metadata (Table Storage)

Tables are stored in `Source.metadata` as structured data:

```go
type TableMetadata struct {
    PageNumber  int                    `json:"page_number"`
    Headers     []string               `json:"headers"`
    Rows        [][]string             `json:"rows"`
    Markdown    string                 `json:"markdown"`      // For search indexing
    BBox        []float64              `json:"bbox"`          // Position on page
}

// In Source.Metadata:
// "tables": [
//   {
//     "page_number": 3,
//     "headers": ["Quarter", "Revenue", "Growth"],
//     "rows": [["Q1", "$1.2M", "15%"], ...],
//     "markdown": "| Quarter | Revenue | Growth |\n|---|---|---|\n| Q1 | $1.2M | 15% |",
//     "bbox": [100, 200, 500, 400]
//   }
// ]
```

---

## Ingestion Flow

### Process Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DOCUMENT INGESTION FLOW                          │
└─────────────────────────────────────────────────────────────────────────┘

    ┌──────────────┐
    │   Document   │
    │  (PDF/DOCX)  │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐
    │  Kreuzberg   │
    │   Parser     │
    └──────┬───────┘
           │
           ▼
    ┌──────────────────────────────────────────────────────────────┐
    │  Parsed Output:                                              │
    │  - content: full text                                        │
    │  - tables[]: structured table data                           │
    │  - images[]: {data: bytes, format, width, height, page_num}  │
    │  - pages[]: {content, images, hierarchy, blocks}             │
    └──────────────────────────────────────────────────────────────┘
           │
           ▼
    ┌──────────────────────────────────────────────────────────────┐
    │                    INGESTION PIPELINE                        │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  1. CREATE SOURCE                                           │
    │     └─ Store metadata, content_type, URI                    │
    │                                                              │
    │  2. PROCESS IMAGES                                          │
    │     ├─ Upload to S3 → get storage_url                       │
    │     └─ Create SourceAsset records with bbox, dimensions     │
    │                                                              │
    │  3. PROCESS TABLES                                          │
    │     ├─ Convert to markdown for search                       │
    │     └─ Store in Source.metadata["tables"]                   │
    │                                                              │
    │  4. CREATE PAGES                                            │
    │     ├─ Store page content                                   │
    │     ├─ Generate ColPali embedding for page image            │
    │     └─ Link to SourceAssets on this page                    │
    │                                                              │
    │  5. CREATE KNOWLEDGE CHUNKS                                 │
    │     ├─ Chunk text content (existing logic)                  │
    │     ├─ Include table markdown in chunks                     │
    │     ├─ Add page_number reference to each chunk              │
    │     └─ Generate text embeddings                             │
    │                                                              │
    └──────────────────────────────────────────────────────────────┘
           │
           ▼
    ┌──────────────────────────────────────────────────────────────┐
    │                    STORAGE LAYERS                            │
    ├──────────────────────────────────────────────────────────────┤
    │                                                              │
    │  PostgreSQL:                    S3:                          │
    │  - sources                     - images/                     │
    │  - pages                         - {source_id}/{page}_{idx}  │
    │  - source_assets                                             │
    │  - knowledge                  │
    │  - knowledge_vectors (pgvector)                             │
    │  - page_vectors (pgvector)                                  │
    │                                                              │
    └──────────────────────────────────────────────────────────────┘
```

### Implementation: Ingestion Service

```go
// internal/core/application/usecases/ingestion/service.go
type IngestionService struct {
    sourceRepo    repositories.SourceRepository
    pageRepo      repositories.PageRepository
    assetRepo     repositories.SourceAssetRepository
    knowledgeRepo repositories.KnowledgeRepository
    assetStorage  storage.AssetStorage           // S3 interface
    textEmbedder  embeddings.TextEmbedder
    pageEmbedder  embeddings.PageEmbedder        // ColPali embedder
    chunker       chunking.Chunker
}

type IngestionResult struct {
    SourceID      uuid.UUID
    PageCount     int
    ImageCount    int
    TableCount    int
    ChunkCount    int
    Errors        []error
}

func (s *IngestionService) IngestDocument(ctx context.Context, req *IngestionRequest) (*IngestionResult, error) {
    // 1. Parse document with kreuzberg
    parsed, err := s.parseDocument(ctx, req)
    if err != nil {
        return nil, err
    }

    // 2. Create source entity
    source := s.createSource(req, parsed)

    // 3. Process and upload images
    assets := s.processImages(ctx, source.ID, parsed.Images)

    // 4. Store table metadata
    source = s.processTables(source, parsed.Tables)

    // 5. Create pages with embeddings
    pages := s.createPages(ctx, source.ID, parsed.Pages, assets)

    // 6. Create knowledge chunks
    chunks := s.createChunks(ctx, source.ID, parsed)

    // 7. Persist everything
    return s.persist(ctx, source, pages, assets, chunks)
}
```

---

## Query Flow

### Search and Retrieval

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          QUERY FLOW                                     │
└─────────────────────────────────────────────────────────────────────────┘

    User Query: "Show me revenue data from Q3"
                    │
                    ▼
    ┌───────────────────────────────────────────┐
    │         TEXT SEARCH (Existing)            │
    │  1. Embed query with text embedder        │
    │  2. Search knowledge_vectors              │
    │  3. Get matching chunks with page_numbers │
    └───────────────────────────────────────────┘
                    │
                    ▼
    ┌───────────────────────────────────────────┐
    │         PAGE RETRIEVAL                    │
    │  1. Get unique page_ids from chunks       │
    │  2. Fetch Page entities with embeddings   │
    │  3. Fetch SourceAssets for each page      │
    └───────────────────────────────────────────┘
                    │
                    ▼
    ┌───────────────────────────────────────────┐
    │         RESPONSE BUILDING                 │
    │  {                                       │
    │    "chunks": [...],  // Matched text      │
    │    "pages": [                            │
    │      {                                   │
    │        "page_number": 3,                 │
    │        "content": "...",                 │
    │        "images": [                       │
    │          {"url": "s3://...", "bbox": []} │
    │        ],                                │
    │        "tables": [...]                   │
    │      }                                   │
    │    ]                                     │
    │  }                                       │
    └───────────────────────────────────────────┘
```

### Visual Similarity Search (Optional Enhancement)

```
    User Query: "Charts showing growth trends"
                    │
                    ▼
    ┌───────────────────────────────────────────┐
    │      VISUAL SEARCH (Page Embeddings)      │
    │  1. Embed query with ColPali embedder     │
    │  2. Search page_vectors                   │
    │  3. Return relevant pages                 │
    └───────────────────────────────────────────┘
```

---

## S3 Storage Structure

```
bucket: eino-notebook-assets/
├── {source_id}/
│   ├── page_1_image_0.png
│   ├── page_1_image_1.png
│   ├── page_3_image_0.png
│   └── ...
└── thumbnails/
    └── {source_id}/
        ├── page_1_image_0_thumb.png
        └── ...
```

---

## Migration Plan

### Phase 1: Database Schema

```sql
-- Migration 000008: Add pages and source_assets tables

-- Create pages table
CREATE TABLE pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    page_number INT NOT NULL,
    content TEXT,
    embedding vector(1024),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id, page_number)
);

CREATE INDEX idx_pages_source ON pages(source_id);
CREATE INDEX idx_pages_embedding ON pages USING hnsw (embedding vector_cosine_ops);

-- Create source_assets table
CREATE TABLE source_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    page_id UUID REFERENCES pages(id) ON DELETE SET NULL,
    asset_type VARCHAR(20) NOT NULL,
    storage_url TEXT NOT NULL,
    bbox FLOAT[],
    width INT,
    height INT,
    format VARCHAR(20),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_assets_source ON source_assets(source_id);
CREATE INDEX idx_assets_page ON source_assets(page_id);
CREATE INDEX idx_assets_type ON source_assets(asset_type);

-- Add page_number to knowledge
ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS page_number INT;
CREATE INDEX IF NOT EXISTS idx_knowledge_page ON knowledge(source_id, page_number);
```

### Phase 2: Repository Interfaces

```go
// internal/core/domain/repositories/page.go
type PageRepository interface {
    Create(ctx context.Context, page *entities.Page) error
    CreateBatch(ctx context.Context, pages []*entities.Page) error
    GetByID(ctx context.Context, id uuid.UUID) (*entities.Page, error)
    GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Page, error)
    GetBySourceAndPage(ctx context.Context, sourceID uuid.UUID, pageNum int) (*entities.Page, error)
    DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error
}

// internal/core/domain/repositories/source_asset.go
type SourceAssetRepository interface {
    Create(ctx context.Context, asset *entities.SourceAsset) error
    CreateBatch(ctx context.Context, assets []*entities.SourceAsset) error
    GetByID(ctx context.Context, id uuid.UUID) (*entities.SourceAsset, error)
    GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.SourceAsset, error)
    GetByPageID(ctx context.Context, pageID uuid.UUID) ([]*entities.SourceAsset, error)
    DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error
}
```

### Phase 3: Storage Interface

```go
// pkg/storage/storage.go
type AssetStorage interface {
    Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)
    UploadBatch(ctx context.Context, items []UploadItem) ([]string, error)
    GetURL(key string) string
    Delete(ctx context.Context, key string) error
    DeleteByPrefix(ctx context.Context, prefix string) error
}

type UploadItem struct {
    Key         string
    Data        []byte
    ContentType string
}

// pkg/storage/s3/storage.go
type S3Storage struct {
    client *s3.Client
    bucket string
    region string
}
```

---

## File Structure

```
internal/
├── core/
│   ├── domain/
│   │   ├── entities/
│   │   │   ├── page.go              # NEW
│   │   │   ├── source_asset.go      # NEW
│   │   │   └── knowledge.go         # MODIFIED (add PageNumber)
│   │   └── repositories/
│   │       ├── page.go              # NEW
│   │       └── source_asset.go      # NEW
│   └── application/
│       ├── usecases/
│       │   └── ingestion/           # NEW
│       │       ├── service.go
│       │       ├── parser.go
│       │       └── chunker.go
│       └── dtos/
│           ├── page.go              # NEW
│           └── source_asset.go      # NEW
├── infrastructure/
│   ├── persistence/
│   │   ├── page.go                  # NEW
│   │   └── source_asset.go          # NEW
│   └── storage/                     # NEW
│       └── s3.go
└── interfaces/
    └── http/
        └── handlers/
            └── ingestion.go         # NEW

pkg/
└── storage/                         # NEW
    ├── storage.go                   # Interface
    └── s3/
        └── storage.go               # S3 implementation

migrations/
└── 000008_add_pages_and_assets.up.sql    # NEW
```

---

## Testing Strategy

### Unit Tests

1. **Entity Tests** - Page and SourceAsset creation, validation
2. **Repository Tests** - CRUD operations with mocks
3. **Ingestion Service Tests** - Pipeline logic with mocked dependencies
4. **Chunker Tests** - Table markdown generation, chunk-to-page mapping

### Integration Tests

1. **Repository Integration** - PostgreSQL operations
2. **S3 Storage Integration** - Upload/retrieve operations
3. **End-to-End Ingestion** - Full pipeline with test documents

### Test Documents

Create sample PDFs with:
- Text only
- Text + images
- Text + tables
- Text + images + tables
- Multi-page with mixed content

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Large documents with many images | Implement batch processing, queue-based ingestion |
| S3 availability issues | Fallback to local storage, retry logic |
| ColPali embedding latency | Async embedding generation, cache results |
| Storage costs grow unbounded | Implement retention policies, cleanup deleted sources |
| Embedding dimension mismatch | Configurable embedding dimensions, version embeddings |

---

## Future Enhancements

1. **OCR on Images** - Extract text from images for better search
2. **Image Captioning** - Generate descriptions for images using VLM
3. **Table QA** - Specialized table question-answering
4. **Document Diff** - Track changes when re-ingesting updated documents
5. **Batch Operations** - Bulk ingest, re-process all sources

---

## Open Questions

1. What ColPali embedding dimension should we use? (1024 is typical)
2. Should we support video/audio assets in the future?
3. What's the expected document size range for planning batch sizes?
4. Do we need to support on-premise deployments (non-S3)?