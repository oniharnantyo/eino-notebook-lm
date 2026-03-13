package usecases

import (
	"context"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/indexer/pgvector"
)

// DocumentUseCase defines the interface for document business logic
// TODO: Implement use case methods based on requirements
type DocumentUseCase interface {
	Create(ctx context.Context, req *dtos.CreateDocumentRequest) (*dtos.DocumentResponse, error)
	GetByID(ctx context.Context, id string) (*dtos.DocumentResponse, error)
	List(ctx context.Context, req *dtos.ListDocumentsRequest) (*dtos.ListDocumentsResponse, error)
	Update(ctx context.Context, req *dtos.UpdateDocumentRequest) (*dtos.DocumentResponse, error)
	Delete(ctx context.Context, id string) error
}

// documentUseCase implements DocumentUseCase
type documentUseCase struct {
	documentRepo repositories.DocumentRepository
	indexer      *pgvector.Indexer
}

// NewDocumentUseCase creates a new document use case
func NewDocumentUseCase(documentRepo repositories.DocumentRepository, indexer *pgvector.Indexer) DocumentUseCase {
	return &documentUseCase{
		documentRepo: documentRepo,
		indexer:      indexer,
	}
}

// Create creates a new document
// TODO: Implement business logic
func (uc *documentUseCase) Create(ctx context.Context, req *dtos.CreateDocumentRequest) (*dtos.DocumentResponse, error) {
	// TODO: Implement document creation logic
	return nil, nil
}

// GetByID retrieves a document by ID
// TODO: Implement business logic
func (uc *documentUseCase) GetByID(ctx context.Context, id string) (*dtos.DocumentResponse, error) {
	// TODO: Implement get by ID logic
	return nil, nil
}

// List retrieves a paginated list of documents
// TODO: Implement business logic
func (uc *documentUseCase) List(ctx context.Context, req *dtos.ListDocumentsRequest) (*dtos.ListDocumentsResponse, error) {
	// TODO: Implement list logic
	return nil, nil
}

// Update updates an existing document
// TODO: Implement business logic
func (uc *documentUseCase) Update(ctx context.Context, req *dtos.UpdateDocumentRequest) (*dtos.DocumentResponse, error) {
	// TODO: Implement update logic
	return nil, nil
}

// Delete deletes a document by ID
// TODO: Implement business logic
func (uc *documentUseCase) Delete(ctx context.Context, id string) error {
	// TODO: Implement delete logic
	return nil
}
