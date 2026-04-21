package sentence

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// SentenceUseCase defines the interface for sentence business logic
type SentenceUseCase interface {
	// ProcessKnowledgeChunks processes knowledge chunks into sentences with embeddings
	ProcessKnowledgeChunks(ctx context.Context, sourceID uuid.UUID, chunks []*entities.Knowledge) error
}

type sentenceUseCase struct {
	sentenceRepo repositories.SentenceRepository
	transformer  document.Transformer
	embedder     embedding.Embedder
}

// NewSentenceUseCase creates a new sentence use case
func NewSentenceUseCase(
	sentenceRepo repositories.SentenceRepository,
	transformer document.Transformer,
	embedder embedding.Embedder,
) SentenceUseCase {
	return &sentenceUseCase{
		sentenceRepo: sentenceRepo,
		transformer:  transformer,
		embedder:     embedder,
	}
}

// ProcessKnowledgeChunks splits knowledge chunks into sentences, embeds them, and saves to repository
func (uc *sentenceUseCase) ProcessKnowledgeChunks(ctx context.Context, sourceID uuid.UUID, chunks []*entities.Knowledge) error {
	for _, chunk := range chunks {
		// Split chunk into smaller units (sentences/sub-chunks)
		docs, err := uc.transformer.Transform(ctx, []*schema.Document{
			{
				ID:      chunk.ID.String(),
				Content: chunk.Content,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to split chunk %s into sentences: %w", chunk.ID, err)
		}

		if len(docs) == 0 {
			continue
		}

		contents := make([]string, len(docs))
		for i, doc := range docs {
			contents[i] = doc.Content
		}

		// Generate embeddings for all units in the chunk
		embeddings, err := uc.embedder.EmbedStrings(ctx, contents)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings for chunk %s: %w", chunk.ID, err)
		}

		sentences := make([]*entities.Sentence, len(docs))
		for i, doc := range docs {
			sentences[i] = &entities.Sentence{
				ID:          uuid.New(),
				KnowledgeID: chunk.ID,
				Content:     doc.Content,
				Embedding:   convertToFloat32(embeddings[i]),
				Position:    i,
				Metadata: map[string]any{
					"source_id": sourceID.String(),
				},
				CreatedAt: time.Now(),
			}
		}

		// Save sentences in batch
		if err := uc.sentenceRepo.SaveBatch(ctx, sentences); err != nil {
			return fmt.Errorf("failed to save sentences for chunk %s: %w", chunk.ID, err)
		}
	}

	return nil
}

func convertToFloat32(v []float64) []float32 {
	res := make([]float32, len(v))
	for i, f := range v {
		res[i] = float32(f)
	}
	return res
}
