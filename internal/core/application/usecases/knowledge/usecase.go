package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// KnowledgeUseCase defines the interface for knowledge business logic
type KnowledgeUseCase interface {
	SaveChunks(ctx context.Context, sourceID uuid.UUID, chunks []kreuzberg.KreuzbergChunk) ([]*entities.Knowledge, error)
}

// knowledgeUseCase implements KnowledgeUseCase
type knowledgeUseCase struct {
	knowledgeRepo repositories.KnowledgeRepository
}

// NewKnowledgeUseCase creates a new knowledge use case
func NewKnowledgeUseCase(
	knowledgeRepo repositories.KnowledgeRepository,
) KnowledgeUseCase {
	return &knowledgeUseCase{
		knowledgeRepo: knowledgeRepo,
	}
}

// SaveChunks converts Kreuzberg chunks to Knowledge entities and saves them in batch
func (uc *knowledgeUseCase) SaveChunks(ctx context.Context, sourceID uuid.UUID, chunks []kreuzberg.KreuzbergChunk) ([]*entities.Knowledge, error) {
	knowledges := make([]*entities.Knowledge, len(chunks))

	for i, chunk := range chunks {
		knowledges[i] = &entities.Knowledge{
			ID:             uuid.New(),
			SourceID:       sourceID,
			Content:        chunk.Content,
			ChunkIndex:     chunk.Metadata.ChunkIndex,
			HeadingContext: chunk.Metadata.HeadingContext,
			FirstPage:      chunk.Metadata.FirstPage,
			LastPage:       chunk.Metadata.LastPage,
			Metadata: map[string]any{
				"total_chunks": chunk.Metadata.TotalChunks,
				"byte_start":   chunk.Metadata.ByteStart,
				"byte_end":     chunk.Metadata.ByteEnd,
				"chunk_type":   chunk.ChunkType,
			},
			CreatedAt: time.Now(),
		}
	}

	if err := uc.knowledgeRepo.SaveBatch(ctx, knowledges); err != nil {
		return nil, fmt.Errorf("failed to save knowledge chunks: %w", err)
	}

	return knowledges, nil
}
