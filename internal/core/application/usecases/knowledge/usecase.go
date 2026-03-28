package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// KnowledgeUseCase defines the interface for knowledge business logic
type KnowledgeUseCase interface {
	Create(ctx context.Context, req *dtos.CreateKnowledgeRequest) error
	GetByID(ctx context.Context, id string) (*dtos.KnowledgeResponse, error)
	List(ctx context.Context, req *dtos.ListKnowledgesRequest) (*dtos.ListKnowledgesResponse, error)
	Update(ctx context.Context, req *dtos.UpdateKnowledgeRequest) (*dtos.KnowledgeResponse, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req *dtos.ListKnowledgesRequest) (*dtos.ListKnowledgesResponse, error)
}

// knowledgeUseCase implements KnowledgeUseCase
type knowledgeUseCase struct {
	knowledgeRepo repositories.KnowledgeRepository
	sourceRepo    repositories.SourceRepository
	indexer       indexer.Indexer
	embedder      embedding.Embedder
	transformer   document.Transformer
}

// NewKnowledgeUseCase creates a new knowledge use case
func NewKnowledgeUseCase(
	knowledgeRepo repositories.KnowledgeRepository,
	sourceRepo repositories.SourceRepository,
	idxr indexer.Indexer,
	embdr embedding.Embedder,
	transformer document.Transformer,
) KnowledgeUseCase {
	return &knowledgeUseCase{
		knowledgeRepo: knowledgeRepo,
		sourceRepo:    sourceRepo,
		indexer:       idxr,
		embedder:      embdr,
		transformer:   transformer,
	}
}

// Create creates a new knowledge from a source and indexes it for search
// This is the main entry point for knowledge ingestion
// It creates knowledge entries that reference an existing source
func (uc *knowledgeUseCase) Create(ctx context.Context, req *dtos.CreateKnowledgeRequest) error {
	// Get source to verify it exists
	source, err := uc.sourceRepo.GetByID(ctx, req.SourceID)
	if err != nil {
		return errors.NewInternalError("failed to find source", err)
	}
	if source == nil {
		return errors.NewNotFoundError("source")
	}

	chunks, err := uc.transformer.Transform(ctx, []*schema.Document{
		{
			ID:       source.ID.String(),
			Content:  req.Content,
			MetaData: req.Metadata,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to transform document: %w", err)
	}

	// Enrich chunks with parent metadata (reference_id, title, source_type, sub_indexes, created_at)
	enrichedChunks := uc.enrichChunksWithParentMetadata(chunks, req, source)

	// Store with embeddings for vector search
	// Use WithSkipExisting to make the operation idempotent - if the same source
	// is processed again (e.g., retry), it will skip existing documents instead of failing
	_, err = uc.indexer.Store(ctx, enrichedChunks,
		indexer.WithEmbedding(uc.embedder),
		pgvector.WithSkipExisting(true),
	)
	if err != nil {
		return fmt.Errorf("failed to index knowledge for search: %v\n", err)
	}

	return nil
}

// sanitizeContent removes null bytes (0x00) from content.
// PostgreSQL rejects null bytes in TEXT columns with error code 22021.
// Null bytes can come from external services like Kreuzberg (OCR artifacts, binary data).
func (uc *knowledgeUseCase) sanitizeContent(content string) string {
	// Fast path: check if null bytes exist
	if !strings.Contains(content, "\x00") {
		return content
	}
	// Remove all null bytes
	return strings.ReplaceAll(content, "\x00", "")
}

// enrichChunksWithParentMetadata adds parent metadata to each chunk
// This ensures that split chunks retain the original document's metadata for filtering and retrieval
func (uc *knowledgeUseCase) enrichChunksWithParentMetadata(chunks []*schema.Document, req *dtos.CreateKnowledgeRequest, source *entities.Source) []*schema.Document {
	enriched := make([]*schema.Document, len(chunks))

	for i, chunk := range chunks {
		// Create a new document to avoid modifying the original chunk
		newDoc := &schema.Document{
			ID:       chunk.ID,
			Content:  chunk.Content,
			MetaData: make(map[string]any),
		}

		// Copy existing metadata from chunk
		if chunk.MetaData != nil {
			for k, v := range chunk.MetaData {
				newDoc.MetaData[k] = v
			}
		}

		// Add parent metadata
		newDoc.MetaData["reference_id"] = req.SourceID.String()

		if req.Title != "" {
			newDoc.MetaData["title"] = req.Title
		}
		if req.SourceType != "" {
			newDoc.MetaData["source_type"] = req.SourceType
		}

		// Merge user-provided metadata
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				newDoc.MetaData[k] = v
			}
		}

		// Add sub_indexes if provided
		if len(req.SubIndexes) > 0 {
			newDoc.MetaData["sub_indexes"] = req.SubIndexes
		}

		// Add source created_at timestamp
		if source != nil && !source.CreatedAt.IsZero() {
			newDoc.MetaData["created_at"] = source.CreatedAt
		}

		enriched[i] = newDoc
	}

	return enriched
}

// mapContentType maps KnowledgeSource to ContentType
func mapContentType(sourceType entities.KnowledgeSource) entities.ContentType {
	switch entityType := sourceType; entityType {
	case entities.SourceDocument:
		return entities.ContentTypePDF
	case entities.SourceWebsite:
		return entities.ContentTypeWebsite
	case entities.SourceText:
		return entities.ContentTypeText
	case entities.SourceAPI:
		return entities.ContentTypeAPI
	default:
		return entities.ContentTypeOther
	}
}

// GetByID retrieves a knowledge by ID
func (uc *knowledgeUseCase) GetByID(ctx context.Context, id string) (*dtos.KnowledgeResponse, error) {
	// Parse ID
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewValidationError("invalid knowledge ID")
	}

	// Find by ID
	knowledge, err := uc.knowledgeRepo.FindByID(ctx, uid)
	if err != nil {
		return nil, errors.NewInternalError("failed to find knowledge", err)
	}
	if knowledge == nil {
		return nil, errors.NewNotFoundError("knowledge")
	}

	return dtos.ToKnowledgeResponse(knowledge), nil
}

// List retrieves a paginated list of knowledges for a source
func (uc *knowledgeUseCase) List(ctx context.Context, req *dtos.ListKnowledgesRequest) (*dtos.ListKnowledgesResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Get knowledges by source ID
	knowledges, err := uc.knowledgeRepo.GetBySourceID(ctx, req.SourceID)
	if err != nil {
		return nil, errors.NewInternalError("failed to list knowledges", err)
	}

	// Apply pagination manually since we're getting all knowledges for the source
	total := len(knowledges)
	start := (req.Page - 1) * req.Limit
	end := start + req.Limit

	if start >= total {
		return &dtos.ListKnowledgesResponse{
			Knowledges: []dtos.KnowledgeResponse{},
			Total:      int64(total),
			Page:       req.Page,
			Limit:      req.Limit,
			TotalPages: 0,
		}, nil
	}

	if end > total {
		end = total
	}

	paginatedKnowledges := knowledges[start:end]

	totalPages := total / req.Limit
	if total%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListKnowledgesResponse{
		Knowledges: dtos.ToKnowledgeResponses(paginatedKnowledges),
		Total:      int64(total),
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing knowledge
func (uc *knowledgeUseCase) Update(ctx context.Context, req *dtos.UpdateKnowledgeRequest) (*dtos.KnowledgeResponse, error) {
	// Check if knowledge exists
	knowledge, err := uc.knowledgeRepo.FindByID(ctx, req.KnowledgeID)
	if err != nil {
		return nil, errors.NewInternalError("failed to find knowledge", err)
	}
	if knowledge == nil {
		return nil, errors.NewNotFoundError("knowledge")
	}

	// Update fields
	if req.Title != "" {
		knowledge.Title = req.Title
	}
	if req.Content != "" {
		knowledge.Content = req.Content
	}
	if req.SourceType != "" {
		knowledge.SourceType = dtos.ParseSourceType(req.SourceType)
	}
	if req.Metadata != nil {
		knowledge.Metadata = req.Metadata
	}
	if req.SubIndexes != nil {
		knowledge.SubIndexes = req.SubIndexes
	}

	// Save to repository
	if err := uc.knowledgeRepo.Save(ctx, knowledge); err != nil {
		return nil, errors.NewInternalError("failed to save knowledge", err)
	}

	return dtos.ToKnowledgeResponse(knowledge), nil
}

// Delete deletes a knowledge by ID
func (uc *knowledgeUseCase) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.NewValidationError("invalid knowledge ID")
	}

	return uc.knowledgeRepo.Delete(ctx, uid)
}

// Search searches knowledges using vector similarity
func (uc *knowledgeUseCase) Search(ctx context.Context, req *dtos.ListKnowledgesRequest) (*dtos.ListKnowledgesResponse, error) {
	if req.Query == "" {
		return uc.List(ctx, req)
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// TODO: Implement vector search using the indexer
	// For now, fall back to listing by source
	return uc.List(ctx, req)
}
