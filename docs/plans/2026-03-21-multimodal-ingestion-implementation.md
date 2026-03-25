# Multimodal Document Ingestion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add support for ingesting tables and images from kreuzberg parser output, enabling rich source preview with visual content while maintaining precise text search capabilities.

**Architecture:** Following Clean Architecture with DDD - new domain entities (Page, SourceAsset), repositories for persistence, S3 storage for binary assets, and updated ingestion pipeline to process multimodal content. Tables stored as markdown in knowledge chunks for search, raw data in source metadata for preview.

**Tech Stack:** Go 1.23, PostgreSQL with pgvector, AWS S3 (or MinIO), kreuzberg HTTP API for document parsing

---

## Design Document Reference

See `docs/plans/2026-03-20-multimodal-ingestion-design.md` for full architectural details.

---

## Task 1: Add PageNumber to Knowledge Entity

**Files:**
- Modify: `internal/core/domain/entities/knowledge.go`

**Step 1: Add PageNumber field to Knowledge struct**

```go
// Add after line 27 (after SourceType field)
PageNumber  *int           `json:"page_number,omitempty" db:"page_number"`  // Reference to source page for context
```

**Step 2: Add getter/setter methods for PageNumber**

```go
// Add at the end of the file (after RemoveSubIndex function)

// SetPageNumber sets the page number reference
func (k *Knowledge) SetPageNumber(pageNum int) {
	k.PageNumber = &pageNum
}

// GetPageNumber returns the page number, returns 0 if nil
func (k *Knowledge) GetPageNumber() int {
	if k.PageNumber == nil {
		return 0
	}
	return *k.PageNumber
}
```

**Step 3: Run existing tests to ensure no breaking changes**

Run: `rtk go test ./internal/core/domain/entities/...`
Expected: PASS (new field is optional, pointer type)

**Step 4: Commit**

```bash
git add internal/core/domain/entities/knowledge.go
git commit -m "feat(knowledge): add page_number field for source page reference"
```

---

## Task 2: Create Page Entity

**Files:**
- Create: `internal/core/domain/entities/page.go`

**Step 1: Write entity test**

Create: `internal/core/domain/entities/page_test.go`

```go
package entities_test

import (
	"testing"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPage(t *testing.T) {
	sourceID := uuid.New()
	pageNum := 1
	content := "Test page content"

	t.Run("creates valid page", func(t *testing.T) {
		page, err := entities.NewPage(sourceID, pageNum, content)

		require.NoError(t, err)
		assert.Equal(t, sourceID, page.SourceID)
		assert.Equal(t, pageNum, page.PageNumber)
		assert.Equal(t, content, page.Content)
		assert.NotEqual(t, uuid.Nil, page.ID)
		assert.False(t, page.CreatedAt.IsZero())
	})

	t.Run("validates required fields", func(t *testing.T) {
		t.Run("empty source ID", func(t *testing.T) {
			_, err := entities.NewPage(uuid.Nil, pageNum, content)
			assert.Error(t, err)
		})

		t.Run("negative page number", func(t *testing.T) {
			_, err := entities.NewPage(sourceID, -1, content)
			assert.Error(t, err)
		})

		t.Run("zero page number", func(t *testing.T) {
			_, err := entities.NewPage(sourceID, 0, content)
			assert.Error(t, err)
		})
	})
}

func TestPage_SetEmbedding(t *testing.T) {
	sourceID := uuid.New()
	page, _ := entities.NewPage(sourceID, 1, "content")

	embedding := []float64{0.1, 0.2, 0.3}
	page.SetEmbedding(embedding)

	assert.Equal(t, embedding, page.Embedding)
	assert.Equal(t, 3, page.EmbeddingDim())
}

func TestPage_Metadata(t *testing.T) {
	sourceID := uuid.New()
	page, _ := entities.NewPage(sourceID, 1, "content")

	t.Run("set and get metadata", func(t *testing.T) {
		page.SetMetadata("width", 612)
		page.SetMetadata("height", 792)
		page.SetMetadata("is_blank", false)

		assert.Equal(t, 612, page.GetMetadata("width"))
		assert.Equal(t, 792, page.GetMetadata("height"))
		assert.Equal(t, false, page.GetMetadata("is_blank"))
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		assert.Nil(t, page.GetMetadata("missing"))
	})
}
```

**Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/core/domain/entities/... -run TestNewPage -v`
Expected: FAIL with "undefined: entities.NewPage"

**Step 3: Implement Page entity**

Create: `internal/core/domain/entities/page.go`

```go
package entities

import (
	"fmt"
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Page represents a single page from a document source
// Pages are used for visual context and page-level embeddings
type Page struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	SourceID  uuid.UUID       `json:"source_id" db:"source_id"`
	PageNumber int            `json:"page_number" db:"page_number"`
	Content   string          `json:"content" db:"content"`
	Embedding []float64       `json:"embedding,omitempty" db:"embedding"` // ColPali-style page embedding
	Metadata  map[string]any  `json:"metadata,omitempty" db:"metadata"`    // width, height, is_blank, block_count
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// NewPage creates a new page entity
func NewPage(sourceID uuid.UUID, pageNum int, content string) (*Page, error) {
	if sourceID.IsEmpty() {
		return nil, fmt.Errorf("source ID cannot be empty")
	}
	if pageNum < 1 {
		return nil, fmt.Errorf("page number must be positive, got %d", pageNum)
	}

	page := &Page{
		ID:        uuid.New(),
		SourceID:  sourceID,
		PageNumber: pageNum,
		Content:   content,
		Metadata:  make(map[string]any),
		CreatedAt: time.Now(),
	}

	return page, nil
}

// SetEmbedding sets the page embedding vector
func (p *Page) SetEmbedding(embedding []float64) {
	p.Embedding = embedding
}

// EmbeddingDim returns the dimension of the embedding vector
func (p *Page) EmbeddingDim() int {
	return len(p.Embedding)
}

// SetMetadata sets a metadata key-value pair
func (p *Page) SetMetadata(key string, value any) {
	if p.Metadata == nil {
		p.Metadata = make(map[string]any)
	}
	p.Metadata[key] = value
}

// GetMetadata gets a metadata value by key
func (p *Page) GetMetadata(key string) any {
	if p.Metadata == nil {
		return nil
	}
	return p.Metadata[key]
}

// IsBlank returns true if the page is blank (no content)
func (p *Page) IsBlank() bool {
	if blank, ok := p.GetMetadata("is_blank").(bool); ok {
		return blank
	}
	return p.Content == ""
}

// Width returns the page width from metadata
func (p *Page) Width() int {
	if w, ok := p.GetMetadata("width").(int); ok {
		return w
	}
	if w, ok := p.GetMetadata("width").(float64); ok {
		return int(w)
	}
	return 0
}

// Height returns the page height from metadata
func (p *Page) Height() int {
	if h, ok := p.GetMetadata("height").(int); ok {
		return h
	}
	if h, ok := p.GetMetadata("height").(float64); ok {
		return int(h)
	}
	return 0
}
```

**Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/core/domain/entities/... -run TestNewPage -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/domain/entities/page.go internal/core/domain/entities/page_test.go
git commit -m "feat(entities): add Page entity for document pages"
```

---

## Task 3: Create SourceAsset Entity

**Files:**
- Create: `internal/core/domain/entities/source_asset.go`
- Create: `internal/core/domain/entities/source_asset_test.go`

**Step 1: Write entity test**

Create: `internal/core/domain/entities/source_asset_test.go`

```go
package entities_test

import (
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSourceAsset(t *testing.T) {
	sourceID := uuid.New()
	assetType := entities.AssetTypeImage
	storageURL := "s3://bucket/source-id/page_1_image_0.png"

	t.Run("creates valid image asset", func(t *testing.T) {
		asset, err := entities.NewSourceAsset(sourceID, assetType, storageURL)

		require.NoError(t, err)
		assert.Equal(t, sourceID, asset.SourceID)
		assert.Equal(t, assetType, asset.AssetType)
		assert.Equal(t, storageURL, asset.StorageURL)
		assert.NotEqual(t, uuid.Nil, asset.ID)
	})

	t.Run("creates valid table asset", func(t *testing.T) {
		asset, err := entities.NewSourceAsset(sourceID, entities.AssetTypeTable, storageURL)

		require.NoError(t, err)
		assert.Equal(t, entities.AssetTypeTable, asset.AssetType)
	})

	t.Run("validates required fields", func(t *testing.T) {
		t.Run("empty source ID", func(t *testing.T) {
			_, err := entities.NewSourceAsset(uuid.Nil, assetType, storageURL)
			assert.Error(t, err)
		})

		t.Run("empty storage URL", func(t *testing.T) {
			_, err := entities.NewSourceAsset(sourceID, assetType, "")
			assert.Error(t, err)
		})

		t.Run("invalid asset type", func(t *testing.T) {
			_, err := entities.NewSourceAsset(sourceID, entities.AssetType("invalid"), storageURL)
			assert.Error(t, err)
		})
	})
}

func TestSourceAsset_SetPageID(t *testing.T) {
	sourceID := uuid.New()
	pageID := uuid.New()
	asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/test.png")

	asset.SetPageID(pageID)

	require.NotNil(t, asset.PageID)
	assert.Equal(t, pageID, *asset.PageID)
}

func TestSourceAsset_SetDimensions(t *testing.T) {
	sourceID := uuid.New()
	asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/test.png")

	asset.SetDimensions(1920, 1080)

	assert.Equal(t, 1920, asset.Width)
	assert.Equal(t, 1080, asset.Height)
}

func TestSourceAsset_SetBBox(t *testing.T) {
	sourceID := uuid.New()
	asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/test.png")

	bbox := []float64{100.0, 200.0, 500.0, 600.0}
	asset.SetBBox(bbox)

	assert.Equal(t, bbox, asset.BBox)
	assert.True(t, asset.HasBBox())
}

func TestSourceAsset_IsImage(t *testing.T) {
	sourceID := uuid.New()

	t.Run("image asset", func(t *testing.T) {
		asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/test.png")
		assert.True(t, asset.IsImage())
		assert.False(t, asset.IsTable())
	})

	t.Run("table asset", func(t *testing.T) {
		asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeTable, "s3://bucket/test.png")
		assert.True(t, asset.IsTable())
		assert.False(t, asset.IsImage())
	})
}
```

**Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/core/domain/entities/... -run TestNewSourceAsset -v`
Expected: FAIL with "undefined: entities.NewSourceAsset"

**Step 3: Implement SourceAsset entity**

Create: `internal/core/domain/entities/source_asset.go`

```go
package entities

import (
	"fmt"
	"time"

	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// AssetType represents the type of asset
type AssetType string

const (
	AssetTypeImage AssetType = "image"
	AssetTypeTable AssetType = "table"
)

// IsValid checks if the asset type is valid
func (a AssetType) IsValid() bool {
	return a == AssetTypeImage || a == AssetTypeTable
}

// SourceAsset represents a binary asset (image, table) from a document source
// Actual binary data is stored in S3, this entity tracks metadata and location
type SourceAsset struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	SourceID   uuid.UUID      `json:"source_id" db:"source_id"`
	PageID     *uuid.UUID     `json:"page_id,omitempty" db:"page_id"`  // Optional link to page
	AssetType  AssetType      `json:"asset_type" db:"asset_type"`
	StorageURL string         `json:"storage_url" db:"storage_url"`    // S3 URL
	BBox       []float64      `json:"bbox,omitempty" db:"bbox"`        // [x1, y1, x2, y2] position on page
	Width      int            `json:"width,omitempty" db:"width"`
	Height     int            `json:"height,omitempty" db:"height"`
	Format     string         `json:"format,omitempty" db:"format"`    // png, jpeg, etc.
	Metadata   map[string]any `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
}

// NewSourceAsset creates a new source asset entity
func NewSourceAsset(sourceID uuid.UUID, assetType AssetType, storageURL string) (*SourceAsset, error) {
	if sourceID.IsEmpty() {
		return nil, fmt.Errorf("source ID cannot be empty")
	}
	if !assetType.IsValid() {
		return nil, fmt.Errorf("invalid asset type: %s", assetType)
	}
	if storageURL == "" {
		return nil, fmt.Errorf("storage URL cannot be empty")
	}

	asset := &SourceAsset{
		ID:         uuid.New(),
		SourceID:   sourceID,
		AssetType:  assetType,
		StorageURL: storageURL,
		Metadata:   make(map[string]any),
		CreatedAt:  time.Now(),
	}

	return asset, nil
}

// SetPageID sets the optional page reference
func (a *SourceAsset) SetPageID(pageID uuid.UUID) {
	a.PageID = &pageID
}

// SetDimensions sets the width and height of the asset
func (a *SourceAsset) SetDimensions(width, height int) {
	a.Width = width
	a.Height = height
}

// SetBBox sets the bounding box coordinates on the page
func (a *SourceAsset) SetBBox(bbox []float64) {
	a.BBox = bbox
}

// HasBBox returns true if bounding box is set
func (a *SourceAsset) HasBBox() bool {
	return len(a.BBox) == 4
}

// SetFormat sets the format of the asset
func (a *SourceAsset) SetFormat(format string) {
	a.Format = format
}

// SetMetadata sets a metadata key-value pair
func (a *SourceAsset) SetMetadata(key string, value any) {
	if a.Metadata == nil {
		a.Metadata = make(map[string]any)
	}
	a.Metadata[key] = value
}

// GetMetadata gets a metadata value by key
func (a *SourceAsset) GetMetadata(key string) any {
	if a.Metadata == nil {
		return nil
	}
	return a.Metadata[key]
}

// IsImage returns true if this is an image asset
func (a *SourceAsset) IsImage() bool {
	return a.AssetType == AssetTypeImage
}

// IsTable returns true if this is a table asset
func (a *SourceAsset) IsTable() bool {
	return a.AssetType == AssetTypeTable
}
```

**Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/core/domain/entities/... -run TestNewSourceAsset -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/domain/entities/source_asset.go internal/core/domain/entities/source_asset_test.go
git commit -m "feat(entities): add SourceAsset entity for images and tables"
```

---

## Task 4: Create Database Migration

**Files:**
- Create: `migrations/000008_add_pages_and_assets.up.sql`

**Step 1: Create migration file**

```sql
-- Migration 000008: Add pages and source_assets tables for multimodal ingestion

-- Enable pgvector extension if not already enabled
CREATE EXTENSION IF NOT EXISTS vector;

-- Create pages table for storing document page content and embeddings
CREATE TABLE IF NOT EXISTS pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    page_number INT NOT NULL,
    content TEXT,
    embedding vector(1024),  -- ColPali-style page embedding
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(source_id, page_number)
);

-- Create indexes for pages
CREATE INDEX IF NOT EXISTS idx_pages_source ON pages(source_id);
CREATE INDEX IF NOT EXISTS idx_pages_embedding ON pages USING hnsw (embedding vector_cosine_ops);

-- Create source_assets table for tracking images and tables stored in S3
CREATE TABLE IF NOT EXISTS source_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    page_id UUID REFERENCES pages(id) ON DELETE SET NULL,
    asset_type VARCHAR(20) NOT NULL,
    storage_url TEXT NOT NULL,
    bbox FLOAT[],  -- [x1, y1, x2, y2] bounding box on page
    width INT,
    height INT,
    format VARCHAR(20),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT check_asset_type CHECK (asset_type IN ('image', 'table'))
);

-- Create indexes for source_assets
CREATE INDEX IF NOT EXISTS idx_assets_source ON source_assets(source_id);
CREATE INDEX IF NOT EXISTS idx_assets_page ON source_assets(page_id);
CREATE INDEX IF NOT EXISTS idx_assets_type ON source_assets(asset_type);

-- Add page_number column to knowledge table for page reference
ALTER TABLE knowledge ADD COLUMN IF NOT EXISTS page_number INT;

-- Create index for knowledge page lookups
CREATE INDEX IF NOT EXISTS idx_knowledge_source_page ON knowledge(source_id, page_number) WHERE page_number IS NOT NULL;

-- Add comment for documentation
COMMENT ON TABLE pages IS 'Stores individual document pages with content and visual embeddings';
COMMENT ON TABLE source_assets IS 'Tracks binary assets (images, tables) stored in S3 with page references';
COMMENT ON COLUMN pages.embedding IS 'ColPali-style page embedding for visual similarity search';
COMMENT ON COLUMN source_assets.bbox IS 'Bounding box coordinates [x1, y1, x2, y2] on source page';
```

**Step 2: Create down migration**

Create: `migrations/000008_add_pages_and_assets.down.sql`

```sql
-- Down migration 000008: Remove pages and source_assets tables

-- Drop indexes first
DROP INDEX IF EXISTS idx_knowledge_source_page;
DROP INDEX IF EXISTS idx_assets_type;
DROP INDEX IF EXISTS idx_assets_page;
DROP INDEX IF EXISTS idx_assets_source;
DROP INDEX IF EXISTS idx_pages_embedding;
DROP INDEX IF EXISTS idx_pages_source;

-- Remove page_number from knowledge
ALTER TABLE knowledge DROP COLUMN IF EXISTS page_number;

-- Drop tables
DROP TABLE IF EXISTS source_assets;
DROP TABLE IF EXISTS pages;
```

**Step 3: Test migration syntax**

Run: `rtk psql -U postgres -d your_database -f migrations/000008_add_pages_and_assets.up.sql`
Expected: SQL executed successfully, tables created

**Step 4: Verify tables created**

Run: `rtk psql -U postgres -d your_database -c "\dt pages" -c "\dt source_assets"`
Expected: Tables listed with correct columns

**Step 5: Commit**

```bash
git add migrations/000008_add_pages_and_assets.up.sql migrations/000008_add_pages_and_assets.down.sql
git commit -m "feat(migrations): add pages and source_assets tables for multimodal support"
```

---

## Task 5: Create Page Repository Interface

**Files:**
- Create: `internal/core/domain/repositories/page.go`

**Step 1: Create repository interface**

```go
package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// PageRepository defines the interface for page persistence operations
type PageRepository interface {
	// Create creates a new page
	Create(ctx context.Context, page *entities.Page) error

	// CreateBatch creates multiple pages in a single transaction
	CreateBatch(ctx context.Context, pages []*entities.Page) error

	// GetByID retrieves a page by ID
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Page, error)

	// GetBySourceID retrieves all pages for a source, ordered by page_number
	GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Page, error)

	// GetBySourceAndPage retrieves a specific page by source and page number
	GetBySourceAndPage(ctx context.Context, sourceID uuid.UUID, pageNum int) (*entities.Page, error)

	// Update updates an existing page (e.g., to set embedding)
	Update(ctx context.Context, page *entities.Page) error

	// UpdateEmbedding updates only the embedding for a page
	UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float64) error

	// DeleteBySourceID deletes all pages for a source
	DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error

	// DeleteByID deletes a single page
	DeleteByID(ctx context.Context, id uuid.UUID) error
}
```

**Step 2: Commit**

```bash
git add internal/core/domain/repositories/page.go
git commit -m "feat(repositories): add PageRepository interface"
```

---

## Task 6: Create SourceAsset Repository Interface

**Files:**
- Create: `internal/core/domain/repositories/source_asset.go`

**Step 1: Create repository interface**

```go
package repositories

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SourceAssetRepository defines the interface for source asset persistence operations
type SourceAssetRepository interface {
	// Create creates a new asset
	Create(ctx context.Context, asset *entities.SourceAsset) error

	// CreateBatch creates multiple assets in a single transaction
	CreateBatch(ctx context.Context, assets []*entities.SourceAsset) error

	// GetByID retrieves an asset by ID
	GetByID(ctx context.Context, id uuid.UUID) (*entities.SourceAsset, error)

	// GetBySourceID retrieves all assets for a source
	GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.SourceAsset, error)

	// GetByPageID retrieves all assets for a specific page
	GetByPageID(ctx context.Context, pageID uuid.UUID) ([]*entities.SourceAsset, error)

	// GetBySourceAndType retrieves assets by source and type
	GetBySourceAndType(ctx context.Context, sourceID uuid.UUID, assetType entities.AssetType) ([]*entities.SourceAsset, error)

	// DeleteBySourceID deletes all assets for a source
	DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error

	// DeleteByID deletes a single asset
	DeleteByID(ctx context.Context, id uuid.UUID) error
}
```

**Step 2: Commit**

```bash
git add internal/core/domain/repositories/source_asset.go
git commit -m "feat(repositories): add SourceAssetRepository interface"
```

---

## Task 7: Implement Page Repository

**Files:**
- Create: `internal/infrastructure/persistence/page.go`
- Create: `internal/infrastructure/persistence/page_test.go`

**Step 1: Write repository test**

Create: `internal/infrastructure/persistence/page_test.go`

```go
package persistence_test

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageRepository_Create(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	t.Run("creates page successfully", func(t *testing.T) {
		page, _ := entities.NewPage(sourceID, 1, "Test page content")

		err := repo.PageRepo.Create(ctx, page)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, page.ID)

		// Verify retrieval
		fetched, err := repo.PageRepo.GetByID(ctx, page.ID)
		require.NoError(t, err)
		assert.Equal(t, page.Content, fetched.Content)
		assert.Equal(t, page.PageNumber, fetched.PageNumber)
	})

	t.Run("prevents duplicate page numbers for same source", func(t *testing.T) {
		page1, _ := entities.NewPage(sourceID, 1, "Content 1")
		page2, _ := entities.NewPage(sourceID, 1, "Content 2")

		err := repo.PageRepo.Create(ctx, page1)
		require.NoError(t, err)

		err = repo.PageRepo.Create(ctx, page2)
		assert.Error(t, err) // UNIQUE constraint violation
	})
}

func TestPageRepository_GetBySourceID(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	// Create test pages
	pages := []*entities.Page{}
	for i := 1; i <= 3; i++ {
		page, _ := entities.NewPage(sourceID, i, fmt.Sprintf("Page %d content", i))
		err := repo.PageRepo.Create(ctx, page)
		require.NoError(t, err)
		pages = append(pages, page)
	}

	t.Run("retrieves all pages ordered by page_number", func(t *testing.T) {
		fetched, err := repo.PageRepo.GetBySourceID(ctx, sourceID)

		require.NoError(t, err)
		assert.Len(t, fetched, 3)
		assert.Equal(t, 1, fetched[0].PageNumber)
		assert.Equal(t, 2, fetched[1].PageNumber)
		assert.Equal(t, 3, fetched[2].PageNumber)
	})
}

func TestPageRepository_UpdateEmbedding(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	page, _ := entities.NewPage(sourceID, 1, "Test content")
	err := repo.PageRepo.Create(ctx, page)
	require.NoError(t, err)

	t.Run("updates embedding successfully", func(t *testing.T) {
		embedding := make([]float64, 1024)
		for i := range embedding {
			embedding[i] = 0.1
		}

		err := repo.PageRepo.UpdateEmbedding(ctx, page.ID, embedding)
		require.NoError(t, err)

		// Verify
		fetched, _ := repo.PageRepo.GetByID(ctx, page.ID)
		assert.Equal(t, embedding, fetched.Embedding)
		assert.Equal(t, 1024, len(fetched.Embedding))
	})
}

func TestPageRepository_DeleteBySourceID(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	// Create pages
	for i := 1; i <= 3; i++ {
		page, _ := entities.NewPage(sourceID, i, fmt.Sprintf("Page %d", i))
		err := repo.PageRepo.Create(ctx, page)
		require.NoError(t, err)
	}

	t.Run("deletes all pages for source", func(t *testing.T) {
		err := repo.PageRepo.DeleteBySourceID(ctx, sourceID)
		require.NoError(t, err)

		fetched, _ := repo.PageRepo.GetBySourceID(ctx, sourceID)
		assert.Len(t, fetched, 0)
	})
}
```

**Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/infrastructure/persistence/... -run TestPageRepository_Create -v`
Expected: FAIL with repository not implemented

**Step 3: Implement Page repository**

Create: `internal/infrastructure/persistence/page.go`

```go
package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// pageRepo implements the PageRepository interface
type pageRepo struct {
	pool *pgxpool.Pool
}

// NewPageRepository creates a new page repository
func NewPageRepository(pool *pgxpool.Pool) repositories.PageRepository {
	return &pageRepo{pool: pool}
}

// Create creates a new page
func (r *pageRepo) Create(ctx context.Context, page *entities.Page) error {
	query := `
		INSERT INTO pages (id, source_id, page_number, content, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		page.ID,
		page.SourceID,
		page.PageNumber,
		page.Content,
		page.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	return nil
}

// CreateBatch creates multiple pages in a single transaction
func (r *pageRepo) CreateBatch(ctx context.Context, pages []*entities.Page) error {
	if len(pages) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO pages (id, source_id, page_number, content, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, page := range pages {
		_, err := tx.Exec(ctx, query,
			page.ID,
			page.SourceID,
			page.PageNumber,
			page.Content,
			page.Metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to create page %d: %w", page.PageNumber, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a page by ID
func (r *pageRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Page, error) {
	query := `
		SELECT id, source_id, page_number, content, embedding, metadata, created_at
		FROM pages
		WHERE id = $1
	`

	var page entities.Page
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&page.ID,
		&page.SourceID,
		&page.PageNumber,
		&page.Content,
		&page.Embedding,
		&page.Metadata,
		&page.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("page not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	return &page, nil
}

// GetBySourceID retrieves all pages for a source, ordered by page_number
func (r *pageRepo) GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.Page, error) {
	query := `
		SELECT id, source_id, page_number, content, embedding, metadata, created_at
		FROM pages
		WHERE source_id = $1
		ORDER BY page_number ASC
	`

	rows, err := r.pool.Query(ctx, query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pages: %w", err)
	}
	defer rows.Close()

	pages := []*entities.Page{}
	for rows.Next() {
		var page entities.Page
		if err := rows.Scan(
			&page.ID,
			&page.SourceID,
			&page.PageNumber,
			&page.Content,
			&page.Embedding,
			&page.Metadata,
			&page.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan page: %w", err)
		}
		pages = append(pages, &page)
	}

	return pages, nil
}

// GetBySourceAndPage retrieves a specific page by source and page number
func (r *pageRepo) GetBySourceAndPage(ctx context.Context, sourceID uuid.UUID, pageNum int) (*entities.Page, error) {
	query := `
		SELECT id, source_id, page_number, content, embedding, metadata, created_at
		FROM pages
		WHERE source_id = $1 AND page_number = $2
	`

	var page entities.Page
	err := r.pool.QueryRow(ctx, query, sourceID, pageNum).Scan(
		&page.ID,
		&page.SourceID,
		&page.PageNumber,
		&page.Content,
		&page.Embedding,
		&page.Metadata,
		&page.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("page not found: source_id=%s page_number=%d", sourceID, pageNum)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	return &page, nil
}

// Update updates an existing page
func (r *pageRepo) Update(ctx context.Context, page *entities.Page) error {
	query := `
		UPDATE pages
		SET content = $2, embedding = $3, metadata = $4
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query,
		page.ID,
		page.Content,
		page.Embedding,
		page.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to update page: %w", err)
	}

	return nil
}

// UpdateEmbedding updates only the embedding for a page
func (r *pageRepo) UpdateEmbedding(ctx context.Context, id uuid.UUID, embedding []float64) error {
	query := `
		UPDATE pages
		SET embedding = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, embedding)
	if err != nil {
		return fmt.Errorf("failed to update page embedding: %w", err)
	}

	return nil
}

// DeleteBySourceID deletes all pages for a source
func (r *pageRepo) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	query := `DELETE FROM pages WHERE source_id = $1`

	_, err := r.pool.Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete pages: %w", err)
	}

	return nil
}

// DeleteByID deletes a single page
func (r *pageRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM pages WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("page not found: %s", id)
	}

	return nil
}
```

**Step 4: Update database.go to wire Page repository**

Modify: `internal/infrastructure/persistence/database.go` (find where repositories are initialized)

Add after sourceRepo initialization:
```go
pageRepo := pageRepo{pool: pool}
```

**Step 5: Run tests to verify they pass**

Run: `rtk go test ./internal/infrastructure/persistence/... -run TestPageRepository -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/infrastructure/persistence/page.go internal/infrastructure/persistence/page_test.go internal/infrastructure/persistence/database.go
git commit -m "feat(persistence): implement PageRepository with PostgreSQL"
```

---

## Task 8: Implement SourceAsset Repository

**Files:**
- Create: `internal/infrastructure/persistence/source_asset.go`
- Create: `internal/infrastructure/persistence/source_asset_test.go`

**Step 1: Write repository test**

Create: `internal/infrastructure/persistence/source_asset_test.go`

```go
package persistence_test

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceAssetRepository_Create(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	t.Run("creates image asset successfully", func(t *testing.T) {
		asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/test.png")

		err := repo.AssetRepo.Create(ctx, asset)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, asset.ID)

		// Verify retrieval
		fetched, err := repo.AssetRepo.GetByID(ctx, asset.ID)
		require.NoError(t, err)
		assert.Equal(t, asset.StorageURL, fetched.StorageURL)
		assert.Equal(t, entities.AssetTypeImage, fetched.AssetType)
	})

	t.Run("creates table asset successfully", func(t *testing.T) {
		asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeTable, "s3://bucket/table.png")

		err := repo.AssetRepo.Create(ctx, asset)

		require.NoError(t, err)
		fetched, _ := repo.AssetRepo.GetByID(ctx, asset.ID)
		assert.Equal(t, entities.AssetTypeTable, fetched.AssetType)
	})
}

func TestSourceAssetRepository_GetBySourceID(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	// Create test assets
	asset1, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/img1.png")
	asset2, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/img2.png")
	repo.AssetRepo.Create(ctx, asset1)
	repo.AssetRepo.Create(ctx, asset2)

	t.Run("retrieves all assets for source", func(t *testing.T) {
		fetched, err := repo.AssetRepo.GetBySourceID(ctx, sourceID)

		require.NoError(t, err)
		assert.Len(t, fetched, 2)
	})

	t.Run("filters by asset type", func(t *testing.T) {
		tableAsset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeTable, "s3://bucket/table.png")
		repo.AssetRepo.Create(ctx, tableAsset)

		images, _ := repo.AssetRepo.GetBySourceAndType(ctx, sourceID, entities.AssetTypeImage)
		tables, _ := repo.AssetRepo.GetBySourceAndType(ctx, sourceID, entities.AssetTypeTable)

		assert.Len(t, images, 2)
		assert.Len(t, tables, 1)
	})
}

func TestSourceAssetRepository_DeleteBySourceID(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)
	sourceID := createTestSource(t, repo.SourceRepo)

	// Create assets
	asset, _ := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "s3://bucket/test.png")
	repo.AssetRepo.Create(ctx, asset)

	t.Run("deletes all assets for source", func(t *testing.T) {
		err := repo.AssetRepo.DeleteBySourceID(ctx, sourceID)
		require.NoError(t, err)

		fetched, _ := repo.AssetRepo.GetBySourceID(ctx, sourceID)
		assert.Len(t, fetched, 0)
	})
}
```

**Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/infrastructure/persistence/... -run TestSourceAssetRepository_Create -v`
Expected: FAIL with repository not implemented

**Step 3: Implement SourceAsset repository**

Create: `internal/infrastructure/persistence/source_asset.go`

```go
package persistence

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// sourceAssetRepo implements the SourceAssetRepository interface
type sourceAssetRepo struct {
	pool *pgxpool.Pool
}

// NewSourceAssetRepository creates a new source asset repository
func NewSourceAssetRepository(pool *pgxpool.Pool) repositories.SourceAssetRepository {
	return &sourceAssetRepo{pool: pool}
}

// Create creates a new asset
func (r *sourceAssetRepo) Create(ctx context.Context, asset *entities.SourceAsset) error {
	query := `
		INSERT INTO source_assets (id, source_id, page_id, asset_type, storage_url, bbox, width, height, format, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		asset.ID,
		asset.SourceID,
		asset.PageID,
		asset.AssetType,
		asset.StorageURL,
		asset.BBox,
		asset.Width,
		asset.Height,
		asset.Format,
		asset.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	return nil
}

// CreateBatch creates multiple assets in a single transaction
func (r *sourceAssetRepo) CreateBatch(ctx context.Context, assets []*entities.SourceAsset) error {
	if len(assets) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO source_assets (id, source_id, page_id, asset_type, storage_url, bbox, width, height, format, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, asset := range assets {
		_, err := tx.Exec(ctx, query,
			asset.ID,
			asset.SourceID,
			asset.PageID,
			asset.AssetType,
			asset.StorageURL,
			asset.BBox,
			asset.Width,
			asset.Height,
			asset.Format,
			asset.Metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to create asset: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves an asset by ID
func (r *sourceAssetRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.SourceAsset, error) {
	query := `
		SELECT id, source_id, page_id, asset_type, storage_url, bbox, width, height, format, metadata, created_at
		FROM source_assets
		WHERE id = $1
	`

	var asset entities.SourceAsset
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&asset.ID,
		&asset.SourceID,
		&asset.PageID,
		&asset.AssetType,
		&asset.StorageURL,
		&asset.BBox,
		&asset.Width,
		&asset.Height,
		&asset.Format,
		&asset.Metadata,
		&asset.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("asset not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	return &asset, nil
}

// GetBySourceID retrieves all assets for a source
func (r *sourceAssetRepo) GetBySourceID(ctx context.Context, sourceID uuid.UUID) ([]*entities.SourceAsset, error) {
	query := `
		SELECT id, source_id, page_id, asset_type, storage_url, bbox, width, height, format, metadata, created_at
		FROM source_assets
		WHERE source_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer rows.Close()

	assets := []*entities.SourceAsset{}
	for rows.Next() {
		var asset entities.SourceAsset
		if err := rows.Scan(
			&asset.ID,
			&asset.SourceID,
			&asset.PageID,
			&asset.AssetType,
			&asset.StorageURL,
			&asset.BBox,
			&asset.Width,
			&asset.Height,
			&asset.Format,
			&asset.Metadata,
			&asset.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetByPageID retrieves all assets for a specific page
func (r *sourceAssetRepo) GetByPageID(ctx context.Context, pageID uuid.UUID) ([]*entities.SourceAsset, error) {
	query := `
		SELECT id, source_id, page_id, asset_type, storage_url, bbox, width, height, format, metadata, created_at
		FROM source_assets
		WHERE page_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer rows.Close()

	assets := []*entities.SourceAsset{}
	for rows.Next() {
		var asset entities.SourceAsset
		if err := rows.Scan(
			&asset.ID,
			&asset.SourceID,
			&asset.PageID,
			&asset.AssetType,
			&asset.StorageURL,
			&asset.BBox,
			&asset.Width,
			&asset.Height,
			&asset.Format,
			&asset.Metadata,
			&asset.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetBySourceAndType retrieves assets by source and type
func (r *sourceAssetRepo) GetBySourceAndType(ctx context.Context, sourceID uuid.UUID, assetType entities.AssetType) ([]*entities.SourceAsset, error) {
	query := `
		SELECT id, source_id, page_id, asset_type, storage_url, bbox, width, height, format, metadata, created_at
		FROM source_assets
		WHERE source_id = $1 AND asset_type = $2
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, sourceID, assetType)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer rows.Close()

	assets := []*entities.SourceAsset{}
	for rows.Next() {
		var asset entities.SourceAsset
		if err := rows.Scan(
			&asset.ID,
			&asset.SourceID,
			&asset.PageID,
			&asset.AssetType,
			&asset.StorageURL,
			&asset.BBox,
			&asset.Width,
			&asset.Height,
			&asset.Format,
			&asset.Metadata,
			&asset.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

// DeleteBySourceID deletes all assets for a source
func (r *sourceAssetRepo) DeleteBySourceID(ctx context.Context, sourceID uuid.UUID) error {
	query := `DELETE FROM source_assets WHERE source_id = $1`

	_, err := r.pool.Exec(ctx, query, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete assets: %w", err)
	}

	return nil
}

// DeleteByID deletes a single asset
func (r *sourceAssetRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM source_assets WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("asset not found: %s", id)
	}

	return nil
}
```

**Step 4: Update database.go to wire SourceAsset repository**

Modify: `internal/infrastructure/persistence/database.go`

Add after pageRepo initialization:
```go
assetRepo := sourceAssetRepo{pool: pool}
```

**Step 5: Run tests to verify they pass**

Run: `rtk go test ./internal/infrastructure/persistence/... -run TestSourceAssetRepository -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/infrastructure/persistence/source_asset.go internal/infrastructure/persistence/source_asset_test.go internal/infrastructure/persistence/database.go
git commit -m "feat(persistence): implement SourceAssetRepository with PostgreSQL"
```

---

## Task 9: Create S3 Storage Interface and Implementation

**Files:**
- Create: `pkg/storage/storage.go`
- Create: `pkg/storage/s3/storage.go`
- Create: `pkg/storage/s3/storage_test.go`

**Step 1: Create storage interface**

Create: `pkg/storage/storage.go`

```go
package storage

import (
	"context"
)

// AssetStorage defines the interface for storing binary assets (images, etc.)
type AssetStorage interface {
	// Upload uploads a single asset and returns the storage URL
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)

	// UploadBatch uploads multiple assets in parallel and returns storage URLs
	UploadBatch(ctx context.Context, items []UploadItem) ([]string, error)

	// GetURL returns the public URL for a given key
	GetURL(key string) string

	// Delete deletes a single asset by key
	Delete(ctx context.Context, key string) error

	// DeleteByPrefix deletes all assets with a given prefix
	DeleteByPrefix(ctx context.Context, prefix string) error
}

// UploadItem represents a single item to upload
type UploadItem struct {
	Key         string
	Data        []byte
	ContentType string
}
```

**Step 2: Write S3 storage test**

Create: `pkg/storage/s3/storage_test.go`

```go
package s3_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3Storage_Upload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	storage := setupTestS3(t)

	t.Run("uploads image successfully", func(t *testing.T) {
		key := "test/image.png"
		data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header

		url, err := storage.Upload(ctx, key, data, "image/png")

		require.NoError(t, err)
		assert.Contains(t, url, key)
	})

	t.Run("uploads batch successfully", func(t *testing.T) {
		items := []storage.UploadItem{
			{Key: "test/img1.png", Data: []byte("img1"), ContentType: "image/png"},
			{Key: "test/img2.png", Data: []byte("img2"), ContentType: "image/png"},
		}

		urls, err := storage.UploadBatch(ctx, items)

		require.NoError(t, err)
		assert.Len(t, urls, 2)
	})

	t.Run("deletes by prefix", func(t *testing.T) {
		// Setup: upload test files
		prefix := "test/delete/"
		items := []storage.UploadItem{
			{Key: prefix + "file1.txt", Data: []byte("content1"), ContentType: "text/plain"},
			{Key: prefix + "file2.txt", Data: []byte("content2"), ContentType: "text/plain"},
		}
		_, _ = storage.UploadBatch(ctx, items)

		// Test delete
		err := storage.DeleteByPrefix(ctx, prefix)
		require.NoError(t, err)
	})
}

func TestS3Storage_GetURL(t *testing.T) {
	storage := setupTestS3(t)

	t.Run("returns proper S3 URL", func(t *testing.T) {
		key := "source-id/page_1_image_0.png"
		url := storage.GetURL(key)

		assert.Contains(t, url, key)
	})
}
```

**Step 3: Implement S3 storage**

Create: `pkg/storage/s3/storage.go`

```go
package s3

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/oniharnantyo/eino-notebook/pkg/storage"
)

// Config holds the S3 storage configuration
type Config struct {
	// Bucket is the S3 bucket name
	Bucket string

	// Region is the AWS region
	Region string

	// Endpoint is the S3 endpoint (for MinIO or other S3-compatible services)
	Endpoint string

	// AccessKey is the AWS access key ID
	AccessKey string

	// SecretKey is the AWS secret access key
	SecretKey string

	// PublicURL is the base URL for public access (optional)
	PublicURL string

	// UsePathStyle indicates whether to use path-style addressing
	UsePathStyle bool
}

// s3Storage implements storage.AssetStorage using AWS S3
type s3Storage struct {
	client    *s3.Client
	config    *Config
	urlPrefix string
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(ctx context.Context, cfg *Config) (storage.AssetStorage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.UsePathStyle
		}
	})

	// Build URL prefix
	urlPrefix := buildURLPrefix(cfg)

	return &s3Storage{
		client:    client,
		config:    cfg,
		urlPrefix: urlPrefix,
	}, nil
}

// Upload uploads a single asset and returns the storage URL
func (s *s3Storage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.config.Bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(string(data)),
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, putInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return s.GetURL(key), nil
}

// UploadBatch uploads multiple assets in parallel
func (s *s3Storage) UploadBatch(ctx context.Context, items []storage.UploadItem) ([]string, error) {
	if len(items) == 0 {
		return []string{}, nil
	}

	results := make([]string, len(items))
	errors := make([]error, len(items))
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, uploadItem storage.UploadItem) {
			defer wg.Done()
			url, err := s.Upload(ctx, uploadItem.Key, uploadItem.Data, uploadItem.ContentType)
			results[idx] = url
			errors[idx] = err
		}(i, item)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("batch upload failed: %w", err)
		}
	}

	return results, nil
}

// GetURL returns the public URL for a given key
func (s *s3Storage) GetURL(key string) string {
	return s.urlPrefix + key
}

// Delete deletes a single asset by key
func (s *s3Storage) Delete(ctx context.Context, key string) error {
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, deleteInput)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// DeleteByPrefix deletes all assets with a given prefix
func (s *s3Storage) DeleteByPrefix(ctx context.Context, prefix string) error {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.config.Bucket),
		Prefix: aws.String(prefix),
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, listInput)

	var objectsToDelete []types.ObjectIdentifier

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	if len(objectsToDelete) == 0 {
		return nil
	}

	// Delete in batches of 1000 (S3 limit)
	for i := 0; i < len(objectsToDelete); i += 1000 {
		end := i + 1000
		if end > len(objectsToDelete) {
			end = len(objectsToDelete)
		}

		batch := objectsToDelete[i:end]
		deleteInput := &s3.DeleteObjectsInput{
			Bucket: aws.String(s.config.Bucket),
			Delete: &types.Delete{
				Objects: batch,
			},
		}

		_, err := s.client.DeleteObjects(ctx, deleteInput)
		if err != nil {
			return fmt.Errorf("failed to delete objects: %w", err)
		}
	}

	return nil
}

// buildURLPrefix constructs the URL prefix for stored objects
func buildURLPrefix(cfg *Config) string {
	if cfg.PublicURL != "" {
		return strings.TrimSuffix(cfg.PublicURL, "/") + "/"
	}

	if cfg.Endpoint != "" {
		// For MinIO or custom endpoints
		return strings.TrimSuffix(cfg.Endpoint, "/") + "/" + cfg.Bucket + "/"
	}

	// Standard S3 URL format
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", cfg.Bucket, cfg.Region)
}
```

**Step 4: Add AWS SDK dependency**

Run: `rtk go get github.com/aws/aws-sdk-go-v2 github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/service/s3`

**Step 5: Run tests (requires S3 or MinIO)**

Run: `rtk go test ./pkg/storage/s3/... -v -short` (skip integration tests)
Or with S3: `rtk go test ./pkg/storage/s3/... -v`

**Step 6: Commit**

```bash
git add pkg/storage/
git commit -m "feat(storage): add S3 storage implementation for binary assets"
```

---

## Task 10: Update Kreuzberg Parser for Page-Level Extraction

**Files:**
- Modify: `pkg/parser/kreuzberg/kreuzberg.go`

**Step 1: Update KreuzbergExtractResponse to include pages**

Add to the struct after line 123:

```go
// Pages contains page-level extraction results
Pages []Page `json:"pages,omitempty"`
```

**Step 2: Add Page struct**

Add after KreuzbergExtractResponse struct:

```go
// Page represents a single page from the document
type Page struct {
	PageNumber int                    `json:"page_number"`
	Content    string                 `json:"content"`
	Images     []Image                `json:"images,omitempty"`
	Tables     []Table                `json:"tables,omitempty"`
	Hierarchy  map[string]interface{} `json:"hierarchy,omitempty"`
	IsBlank    bool                   `json:"is_blank"`
}

// Image represents an extracted image
type Image struct {
	Data        []byte  `json:"data"`
	Format      string  `json:"format"`
	ImageIndex  int     `json:"image_index"`
	PageNumber  int     `json:"page_number"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Colorspace  string  `json:"colorspace"`
	BitsPerComponent int `json:"bits_per_component"`
	IsMask      bool    `json:"is_mask"`
}

// Table represents an extracted table
type Table struct {
	PageNumber int      `json:"page_number"`
	Headers    []string `json:"headers,omitempty"`
	Rows       [][]string `json:"rows,omitempty"`
	Markdown   string   `json:"markdown,omitempty"`
	BBox       []float64 `json:"bbox,omitempty"`
}
```

**Step 3: Update Parse to return pages**

Modify the Parse function to populate Pages from the response. Add after line 227:

```go
// Store page-level data if available
if len(result.Pages) > 0 {
	resultMeta["page_count"] = len(result.Pages)
}
```

**Step 4: Commit**

```bash
git add pkg/parser/kreuzberg/kreuzberg.go
git commit -m "feat(parser): update kreuzberg parser for page-level extraction"
```

---

## Task 11: Create Ingestion Service

**Files:**
- Create: `internal/core/application/usecases/ingestion/service.go`
- Create: `internal/core/application/usecases/ingestion/service_test.go`

**Step 1: Write service test**

Create: `internal/core/application/usecases/ingestion/service_test.go`

```go
package ingestion_test

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/ingestion"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestionService_IngestDocument(t *testing.T) {
	ctx := context.Background()
	notebookID := uuid.New()
	service := setupIngestionService(t)

	t.Run("ingests PDF with images and tables", func(t *testing.T) {
		// Sample kreuzberg response
		kreuzbergResp := `[
			{
				"content": "Document content",
				"pages": [
					{
						"page_number": 1,
						"content": "Page 1 content",
						"images": [
							{
								"data": [137, 80, 78, 71],
								"format": "FlateDecode",
								"image_index": 0,
								"page_number": 1,
								"width": 100,
								"height": 100
							}
						],
						"hierarchy": {"block_count": 2},
						"is_blank": false
					}
				],
				"tables": [
					{
						"page_number": 1,
						"headers": ["Col1", "Col2"],
						"rows": [["A", "B"]],
						"markdown": "| Col1 | Col2 |\n|---|---|\n| A | B |"
					}
				],
				"metadata": {"page_count": 1}
			}
		]`

		req := &ingestion.IngestDocumentRequest{
			NotebookID:     notebookID,
			Title:          "Test Document.pdf",
			URI:            "file:///test.pdf",
			ContentType:    "application/pdf",
			KreuzbergResp:  []byte(kreuzbergResp),
		}

		result, err := service.IngestDocument(ctx, req)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.SourceID)
		assert.Equal(t, 1, result.PageCount)
		assert.Equal(t, 1, result.ImageCount)
		assert.Equal(t, 1, result.TableCount)
		assert.Greater(t, result.ChunkCount, 0)
	})
}
```

**Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/core/application/usecases/ingestion/... -run TestIngestionService_IngestDocument -v`
Expected: FAIL with service not implemented

**Step 3: Implement ingestion service**

Create: `internal/core/application/usecases/ingestion/service.go`

```go
package ingestion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/storage"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// Service handles document ingestion with multimodal content
type Service struct {
	sourceRepo    repositories.SourceRepository
	pageRepo      repositories.PageRepository
	assetRepo     repositories.SourceAssetRepository
	knowledgeRepo repositories.KnowledgeRepository
	assetStorage  storage.AssetStorage
	chunker       Chunker
}

// Chunker defines the interface for chunking document content
type Chunker interface {
	Chunk(ctx context.Context, content string) ([]string, error)
}

// NewService creates a new ingestion service
func NewService(
	sourceRepo repositories.SourceRepository,
	pageRepo repositories.PageRepository,
	assetRepo repositories.SourceAssetRepository,
	knowledgeRepo repositories.KnowledgeRepository,
	assetStorage storage.AssetStorage,
	chunker Chunker,
) *Service {
	return &Service{
		sourceRepo:    sourceRepo,
		pageRepo:      pageRepo,
		assetRepo:     assetRepo,
		knowledgeRepo: knowledgeRepo,
		assetStorage:  assetStorage,
		chunker:       chunker,
	}
}

// IngestDocumentRequest defines the request for document ingestion
type IngestDocumentRequest struct {
	NotebookID    uuid.UUID
	Title         string
	URI           string
	ContentType   entities.ContentType
	KreuzbergResp []byte
}

// IngestDocumentResult defines the result of document ingestion
type IngestDocumentResult struct {
	SourceID   uuid.UUID
	PageCount  int
	ImageCount int
	TableCount int
	ChunkCount int
	Errors     []error
}

// IngestDocument ingests a document with multimodal content
func (s *Service) IngestDocument(ctx context.Context, req *IngestDocumentRequest) (*IngestDocumentResult, error) {
	// 1. Parse kreuzberg response
	var kreuzbergResults []kreuzberg.KreuzbergExtractResponse
	if err := json.Unmarshal(req.KreuzbergResp, &kreuzbergResults); err != nil {
		return nil, fmt.Errorf("failed to parse kreuzberg response: %w", err)
	}
	if len(kreuzbergResults) == 0 {
		return nil, fmt.Errorf("empty kreuzberg response")
	}

	result := &kreuzbergResults[0]

	// 2. Create source entity
	source, err := entities.NewSource(req.NotebookID, req.Title, req.URI, req.ContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}
	source.SetContent(result.Content, len(result.Content))

	// Store table metadata in source metadata
	if len(result.Tables) > 0 {
		var tables []interface{}
		for _, t := range result.Tables {
			if tableMap, ok := t.(map[string]interface{}); ok {
				tables = append(tables, tableMap)
			}
		}
		source.SetMetadata("tables", tables)
	}

	// Store page count
	if pageCount, ok := result.Metadata["page_count"].(float64); ok {
		source.SetMetadata("page_count", int(pageCount))
	}

	if err := s.sourceRepo.Create(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to create source: %w", err)
	}

	ingestionResult := &IngestDocumentResult{
		SourceID:   source.ID,
		PageCount:  0,
		ImageCount: 0,
		TableCount: len(result.Tables),
	}

	// 3. Process pages and images
	if len(result.Pages) > 0 {
		pages, assets, err := s.processPages(ctx, source.ID, result.Pages)
		if err != nil {
			ingestionResult.Errors = append(ingestionResult.Errors, err)
		} else {
			ingestionResult.PageCount = len(pages)
			ingestionResult.ImageCount = len(assets)
		}
	}

	// 4. Create knowledge chunks
	chunks, err := s.chunker.Chunk(ctx, result.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk content: %w", err)
	}

	// Create knowledge entities
	for _, chunkContent := range chunks {
		knowledge, err := entities.NewKnowledge(
			source.ID,
			req.Title,
			chunkContent,
			entities.SourceDocument,
			map[string]any{
				"extractor": "kreuzberg",
			},
		)
		if err != nil {
			ingestionResult.Errors = append(ingestionResult.Errors, err)
			continue
		}

		if err := s.knowledgeRepo.Create(ctx, knowledge); err != nil {
			ingestionResult.Errors = append(ingestionResult.Errors, err)
			continue
		}

		source.IncrementChunkCount()
		ingestionResult.ChunkCount++
	}

	// Update source chunk count
	if err := s.sourceRepo.Update(ctx, source); err != nil {
		ingestionResult.Errors = append(ingestionResult.Errors, err)
	}

	return ingestionResult, nil
}

// processPages processes pages from kreuzberg response
func (s *Service) processPages(ctx context.Context, sourceID uuid.UUID, pagesData []interface{}) ([]*entities.Page, []*entities.SourceAsset, error) {
	var pages []*entities.Page
	var assets []*entities.SourceAsset

	for _, pageData := range pagesData {
		pageMap, ok := pageData.(map[string]interface{})
		if !ok {
			continue
		}

		pageNumFloat, ok := pageMap["page_number"].(float64)
		if !ok {
			continue
		}
		pageNum := int(pageNumFloat)

		content, _ := pageMap["content"].(string)
		isBlank, _ := pageMap["is_blank"].(bool)
		hierarchy, _ := pageMap["hierarchy"].(map[string]interface{})

		// Create page entity
		page, err := entities.NewPage(sourceID, pageNum, content)
		if err != nil {
			return nil, nil, err
		}

		if isBlank {
			page.SetMetadata("is_blank", true)
		}
		if blockCount, ok := hierarchy["block_count"].(float64); ok {
			page.SetMetadata("block_count", int(blockCount))
		}

		pages = append(pages, page)

		// Process images on this page
		if imagesData, ok := pageMap["images"].([]interface{}); ok {
			pageAssets, err := s.processImages(ctx, sourceID, page.ID, pageNum, imagesData)
			if err != nil {
				return nil, nil, err
			}
			assets = append(assets, pageAssets...)
		}
	}

	// Batch create pages
	if err := s.pageRepo.CreateBatch(ctx, pages); err != nil {
		return nil, nil, fmt.Errorf("failed to create pages: %w", err)
	}

	// Batch create assets
	if len(assets) > 0 {
		if err := s.assetRepo.CreateBatch(ctx, assets); err != nil {
			return nil, nil, fmt.Errorf("failed to create assets: %w", err)
		}
	}

	return pages, assets, nil
}

// processImages processes images from a page
func (s *Service) processImages(ctx context.Context, sourceID, pageID uuid.UUID, pageNum int, imagesData []interface{}) ([]*entities.SourceAsset, error) {
	var assets []*entities.SourceAsset
	var uploadItems []storage.UploadItem

	// First, upload all images to S3
	for i, imgData := range imagesData {
		imgMap, ok := imgData.(map[string]interface{})
		if !ok {
			continue
		}

		dataBytes, ok := imgMap["data"].([]interface{})
		if !ok {
			continue
		}

		// Convert []interface{} to []byte
		data := make([]byte, len(dataBytes))
		for j, b := range dataBytes {
			if bFloat, ok := b.(float64); ok {
				data[j] = byte(bFloat)
			}
		}

		// Generate storage key
		key := fmt.Sprintf("%s/page_%d_image_%d.png", sourceID, pageNum, i)

		format, _ := imgMap["format"].(string)
		width, _ := imgMap["width"].(float64)
		height, _ := imgMap["height"].(float64)

		uploadItems = append(uploadItems, storage.UploadItem{
			Key:         key,
			Data:        data,
			ContentType: "image/png",
		})

		// Create asset entity
		asset, err := entities.NewSourceAsset(sourceID, entities.AssetTypeImage, "")
		if err != nil {
			return nil, err
		}

		asset.SetPageID(pageID)
		asset.SetDimensions(int(width), int(height))
		asset.SetFormat(format)

		assets = append(assets, asset)
	}

	// Batch upload to S3
	if len(uploadItems) > 0 {
		urls, err := s.assetStorage.UploadBatch(ctx, uploadItems)
		if err != nil {
			return nil, fmt.Errorf("failed to upload images: %w", err)
		}

		// Update storage URLs
		for i, url := range urls {
			assets[i].StorageURL = url
		}
	}

	return assets, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `rtk go test ./internal/core/application/usecases/ingestion/... -run TestIngestionService_IngestDocument -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/application/usecases/ingestion/
git commit -m "feat(ingestion): add ingestion service for multimodal content"
```

---

## Task 12: Update Knowledge Repository for Page Number Support

**Files:**
- Modify: `internal/infrastructure/persistence/knowledge.go`

**Step 1: Update Create method to include page_number**

Find the INSERT query in Create method and add page_number column:

```go
query := `
	INSERT INTO knowledge (knowledge_id, source_id, title, content, source_type, metadata, sub_indexes, page_number)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
```

Update the Exec call to include page_number:
```go
_, err := r.pool.Exec(ctx, query,
	knowledge.KnowledgeID,
	knowledge.SourceID,
	knowledge.Title,
	knowledge.Content,
	knowledge.SourceType,
	knowledge.Metadata,
	knowledge.SubIndexes,
	knowledge.PageNumber,
)
```

**Step 2: Update GetBySourceID to include page_number**

Add page_number to SELECT query:
```go
query := `
	SELECT knowledge_id, source_id, title, content, source_type, metadata, sub_indexes, created_at, page_number
	FROM knowledge
	WHERE source_id = $1
	ORDER BY created_at DESC
`
```

Update the Scan call:
```go
if err := rows.Scan(
	&knowledge.KnowledgeID,
	&knowledge.SourceID,
	&knowledge.Title,
	&knowledge.Content,
	&knowledge.SourceType,
	&knowledge.Metadata,
	&knowledge.SubIndexes,
	&knowledge.CreatedAt,
	&knowledge.PageNumber,
); err != nil {
```

**Step 3: Run existing tests**

Run: `rtk go test ./internal/infrastructure/persistence/... -run TestKnowledgeRepository -v`
Expected: PASS (backwards compatible, page_number is nullable)

**Step 4: Commit**

```bash
git add internal/infrastructure/persistence/knowledge.go
git commit -m "feat(persistence): add page_number support to knowledge repository"
```

---

## Task 13: Wire Up Ingestion Service in Application

**Files:**
- Modify: `internal/infrastructure/wire/wire.go` (or wherever dependency injection is done)

**Step 1: Add ingestion service to wire.go**

```go
// Add ingestion service
ingestionService := ingestion.NewService(
	repositories.Source,
	repositories.Page,
	repositories.SourceAsset,
	repositories.Knowledge,
	storage.S3,
	chunker.NewChunker(),  // or however chunker is initialized
)
```

**Step 2: Commit**

```bash
git add internal/infrastructure/wire/wire.go
git commit -m "feat(wire): wire up ingestion service"
```

---

## Task 14: Create Ingestion HTTP Handler

**Files:**
- Create: `internal/interfaces/http/handlers/ingestion.go`
- Create: `internal/interfaces/http/handlers/ingestion_test.go`

**Step 1: Write handler test**

Create: `internal/interfaces/http/handlers/ingestion_test.go`

```go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/interfaces/http/handlers"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestionHandler_IngestDocument(t *testing.T) {
	ctx := context.Background()
	app := setupTestApp(t)

	t.Run("ingests document successfully", func(t *testing.T) {
		notebookID := uuid.New()

		// Sample kreuzberg response
		kreuzbergResp := map[string]interface{}{
			"pages": []map[string]interface{}{
				{
					"page_number": float64(1),
					"content":     "Test page content",
					"images":      []interface{}{},
					"is_blank":    false,
				},
			},
			"tables": []interface{}{},
		}
		respBody, _ := json.Marshal(kreuzbergResp)

		reqBody := map[string]interface{}{
			"notebook_id":     notebookID.String(),
			"title":           "Test Document.pdf",
			"uri":             "file:///test.pdf",
			"content_type":    "application/pdf",
			"kreuzberg_resp":  string(respBody),
		}
		reqBodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/ingest", bytes.NewReader(reqBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		app.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp handlers.IngestDocumentResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, resp.SourceID)
		assert.Equal(t, 1, resp.PageCount)
	})
}
```

**Step 2: Run test to verify it fails**

Run: `rtk go test ./internal/interfaces/http/handlers/... -run TestIngestionHandler_IngestDocument -v`
Expected: FAIL with handler not implemented

**Step 3: Implement handler**

Create: `internal/interfaces/http/handlers/ingestion.go`

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/ingestion"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// IngestionHandler handles document ingestion requests
type IngestionHandler struct {
	service *ingestion.Service
}

// NewIngestionHandler creates a new ingestion handler
func NewIngestionHandler(service *ingestion.Service) *IngestionHandler {
	return &IngestionHandler{
		service: service,
	}
}

// IngestDocumentRequest defines the HTTP request for document ingestion
type IngestDocumentRequest struct {
	NotebookID   string `json:"notebook_id" binding:"required"`
	Title        string `json:"title" binding:"required"`
	URI          string `json:"uri" binding:"required"`
	ContentType  string `json:"content_type" binding:"required"`
	KreuzbergResp string `json:"kreuzberg_resp" binding:"required"`
}

// IngestDocumentResponse defines the HTTP response
type IngestDocumentResponse struct {
	SourceID   uuid.UUID `json:"source_id"`
	PageCount  int       `json:"page_count"`
	ImageCount int       `json:"image_count"`
	TableCount int       `json:"table_count"`
	ChunkCount int       `json:"chunk_count"`
}

// IngestDocument handles POST /api/v1/ingest
func (h *IngestionHandler) IngestDocument(c *gin.Context) {
	var req IngestDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse notebook ID
	notebookID, err := uuid.Parse(req.NotebookID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid notebook_id"})
		return
	}

	// Parse content type
	contentType := entities.ContentType(req.ContentType)

	// Call ingestion service
	result, err := h.service.IngestDocument(c.Request.Context(), &ingestion.IngestDocumentRequest{
		NotebookID:   notebookID,
		Title:        req.Title,
		URI:          req.URI,
		ContentType:  contentType,
		KreuzbergResp: []byte(req.KreuzbergResp),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, IngestDocumentResponse{
		SourceID:   result.SourceID,
		PageCount:  result.PageCount,
		ImageCount: result.ImageCount,
		TableCount: result.TableCount,
		ChunkCount: result.ChunkCount,
	})
}
```

**Step 4: Register route**

Modify: `internal/interfaces/http/routes/routes.go` (or wherever routes are defined)

```go
// Add to router setup
ingestionHandler := handlers.NewIngestionHandler(services.Ingestion)
v1.POST("/ingest", ingestionHandler.IngestDocument)
```

**Step 5: Run tests to verify they pass**

Run: `rtk go test ./internal/interfaces/http/handlers/... -run TestIngestionHandler_IngestDocument -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/interfaces/http/handlers/ingestion.go internal/interfaces/http/routes/routes.go
git commit -m "feat(http): add ingestion endpoint for multimodal documents"
```

---

## Task 15: End-to-End Integration Test

**Files:**
- Create: `tests/integration/multimodal_ingestion_test.go`

**Step 1: Create integration test**

```go
package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/ingestion"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/storage"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultimodalIngestion_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	app := setupIntegrationTest(t)

	// Load test kreuzberg response
	kreuzbergData, err := os.ReadFile("testdata/kreuzberg_response.json")
	require.NoError(t, err)

	t.Run("full ingestion workflow", func(t *testing.T) {
		notebookID := createTestNotebook(t, ctx, app)

		req := &ingestion.IngestDocumentRequest{
			NotebookID:    notebookID,
			Title:         "Test Document.pdf",
			URI:           "file:///test.pdf",
			ContentType:   entities.ContentTypePDF,
			KreuzbergResp: kreuzbergData,
		}

		result, err := app.IngestionService.IngestDocument(ctx, req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.SourceID)

		// Verify pages created
		pages, err := app.PageRepo.GetBySourceID(ctx, result.SourceID)
		require.NoError(t, err)
		assert.Greater(t, len(pages), 0)

		// Verify assets created
		assets, err := app.AssetRepo.GetBySourceID(ctx, result.SourceID)
		require.NoError(t, err)
		assert.Greater(t, len(assets), 0)

		// Verify knowledge chunks created
		knowledgeList, _, err := app.KnowledgeRepo.ListBySourceID(ctx, repositories.KnowledgeFilter{
			SourceID: &result.SourceID,
		})
		require.NoError(t, err)
		assert.Greater(t, len(knowledgeList), 0)

		// Verify source metadata has tables
		source, err := app.SourceRepo.GetByID(ctx, result.SourceID)
		require.NoError(t, err)

		tables, ok := source.Metadata["tables"]
		assert.True(t, ok)
		assert.NotNil(t, tables)
	})
}
```

**Step 2: Run integration test**

Run: `rtk go test ./tests/integration/... -run TestMultimodalIngestion_FullWorkflow -v`

**Step 3: Commit**

```bash
git add tests/integration/multimodal_ingestion_test.go
git commit -m "test(integration): add full workflow test for multimodal ingestion"
```

---

## Task 16: Documentation

**Files:**
- Create: `docs/features/multimodal-ingestion.md`

**Step 1: Create feature documentation**

```markdown
# Multimodal Document Ingestion

## Overview

The application supports ingesting documents with rich multimodal content including images and tables. Documents are parsed using the Kreuzberg service and stored with page-level granularity for rich preview and retrieval.

## Supported Content Types

- **PDF** - Full support for text, images, and tables
- **DOCX** - Text extraction (coming soon)
- **Images** - OCR extraction (coming soon)

## Ingestion Flow

1. Upload document via API
2. Kreuzberg service parses the document
3. Images are extracted and uploaded to S3
4. Tables are converted to markdown and stored as structured data
5. Content is chunked for vector search
6. Pages are created with content references

## API Usage

```bash
curl -X POST http://localhost:8080/api/v1/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "notebook_id": "uuid-here",
    "title": "Document.pdf",
    "uri": "file:///path/to/doc.pdf",
    "content_type": "application/pdf",
    "kreuzberg_resp": "{...}"
  }'
```

## Storage

- **PostgreSQL** - Sources, pages, knowledge chunks, metadata
- **pgvector** - Text and page embeddings
- **S3** - Binary assets (images, page screenshots)
```

**Step 2: Commit**

```bash
git add docs/features/multimodal-ingestion.md
git commit -m "docs: add multimodal ingestion feature documentation"
```

---

## Verification Steps

After completing all tasks:

1. **Run all tests**
   ```bash
   rtk go test ./... -v
   ```

2. **Run migration**
   ```bash
   rtk psql -U postgres -d your_database -f migrations/000008_add_pages_and_assets.up.sql
   ```

3. **Test API endpoint**
   ```bash
   curl -X POST http://localhost:8080/api/v1/ingest -d @test_request.json
   ```

4. **Verify database**
   ```bash
   rtk psql -U postgres -d your_database -c "SELECT COUNT(*) FROM pages"
   rtk psql -U postgres -d your_database -c "SELECT COUNT(*) FROM source_assets"
   ```

5. **Verify S3 storage**
   ```bash
   aws s3 ls s3://your-bucket/
   ```

---

## Implementation Notes

1. **Page Embeddings** - Currently stored but not generated. Use ColPali or similar for visual embeddings.
2. **Async Processing** - For large documents, consider implementing queue-based ingestion.
3. **Cleanup** - Implement cascade deletion cleanup for S3 assets when source is deleted.
4. **Error Handling** - Partial failure handling could be improved (e.g., some images fail to upload).

---

## Dependencies Added

- `github.com/aws/aws-sdk-go-v2`
- `github.com/aws/aws-sdk-go-v2/config`
- `github.com/aws/aws-sdk-go-v2/service/s3`

Ensure these are added to `go.mod`.