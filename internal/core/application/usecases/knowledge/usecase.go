package knowledge

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/errors"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
)

// KnowledgeUseCase defines the interface for knowledge business logic
type KnowledgeUseCase interface {
	Create(ctx context.Context, req *dtos.CreateKnowledgeRequest) error
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
