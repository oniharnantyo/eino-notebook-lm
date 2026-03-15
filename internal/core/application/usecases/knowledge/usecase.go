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
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// KnowledgeUseCase defines the interface for knowledge business logic
type KnowledgeUseCase interface {
	Create(ctx context.Context, req *dtos.CreateKnowledgeRequest) (*dtos.KnowledgeResponse, error)
	GetByID(ctx context.Context, id string) (*dtos.KnowledgeResponse, error)
	List(ctx context.Context, req *dtos.ListKnowledgesRequest) (*dtos.ListKnowledgesResponse, error)
	Update(ctx context.Context, req *dtos.UpdateKnowledgeRequest) (*dtos.KnowledgeResponse, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req *dtos.ListKnowledgesRequest) (*dtos.ListKnowledgesResponse, error)
}

// knowledgeUseCase implements KnowledgeUseCase
type knowledgeUseCase struct {
	knowledgeRepo repositories.KnowledgeRepository
	indexer       indexer.Indexer
	embedder      embedding.Embedder
	transformer   document.Transformer
}

// NewKnowledgeUseCase creates a new knowledge use case
func NewKnowledgeUseCase(knowledgeRepo repositories.KnowledgeRepository, idxr indexer.Indexer, embdr embedding.Embedder, transformer document.Transformer) KnowledgeUseCase {
	return &knowledgeUseCase{
		knowledgeRepo: knowledgeRepo,
		indexer:       idxr,
		embedder:      embdr,
		transformer:   transformer,
	}
}

// Create creates a new knowledge and indexes it for search
func (uc *knowledgeUseCase) Create(ctx context.Context, req *dtos.CreateKnowledgeRequest) (*dtos.KnowledgeResponse, error) {
	// Create the entity
	sourceType := dtos.ParseSourceType(req.SourceType)
	knowledge, err := entities.NewKnowledge(req.NotebookID, req.Title, req.Content, sourceType, req.Metadata)
	if err != nil {
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create knowledge: %v", err))
	}

	// Add sub-indexes if provided
	for _, idx := range req.SubIndexes {
		knowledge.AddSubIndex(idx)
	}

	// Index for vector search if embedder is available
	// Create document for indexing
	doc := &schema.Document{
		ID:      uuid.New().String(),
		Content: knowledge.Content,
		MetaData: map[string]any{
			"title":        knowledge.Title,
			"reference_id": knowledge.NotebookID.String(),
			"source_type":  knowledge.SourceType,
			"created_at":   knowledge.CreatedAt,
		},
	}

	// Add sub-indexes to metadata
	if len(knowledge.SubIndexes) > 0 {
		doc.MetaData["sub_indexes"] = knowledge.SubIndexes
	}

	splitDocs, err := uc.transformer.Transform(ctx, []*schema.Document{doc})
	if err != nil {
		return nil, fmt.Errorf("failed to transform document: %v", err)
	}

	// Store with embeddings
	_, err = uc.indexer.Store(ctx, splitDocs, indexer.WithEmbedding(uc.embedder))
	if err != nil {
		return nil, fmt.Errorf("failed to index knowledge %s for search: %v\n", knowledge.KnowledgeID, err)
	}

	return dtos.ToKnowledgeResponse(knowledge), nil
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

// List retrieves a paginated list of knowledges for a notebook
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

	offset := (req.Page - 1) * req.Limit

	var knowledges []*entities.Knowledge
	var err error
	var total int64

	// Filter by source type if provided
	if req.SourceType != "" {
		knowledges, err = uc.knowledgeRepo.FindByNotebookIDAndSourceType(ctx, req.NotebookID, req.SourceType, req.Limit, offset)
		total, _ = uc.knowledgeRepo.CountByNotebookID(ctx, req.NotebookID)
	} else {
		knowledges, err = uc.knowledgeRepo.FindByNotebookID(ctx, req.NotebookID, req.Limit, offset)
		total, err = uc.knowledgeRepo.CountByNotebookID(ctx, req.NotebookID)
	}

	if err != nil {
		return nil, errors.NewInternalError("failed to list knowledges", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListKnowledgesResponse{
		Knowledges: dtos.ToKnowledgeResponses(knowledges),
		Total:      total,
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

	offset := (req.Page - 1) * req.Limit

	// TODO: Implement vector search using the indexer
	// For now, fall back to repository search
	var knowledges []*entities.Knowledge
	var err error
	var total int64

	knowledges, err = uc.knowledgeRepo.FindByNotebookID(ctx, req.NotebookID, req.Limit, offset)
	total, _ = uc.knowledgeRepo.CountByNotebookID(ctx, req.NotebookID)

	if err != nil {
		return nil, errors.NewInternalError("failed to search knowledges", err)
	}

	totalPages := int(total) / req.Limit
	if int(total)%req.Limit > 0 {
		totalPages++
	}

	return &dtos.ListKnowledgesResponse{
		Knowledges: dtos.ToKnowledgeResponses(knowledges),
		Total:      total,
		Page:       req.Page,
		Limit:      req.Limit,
		TotalPages: totalPages,
	}, nil
}
