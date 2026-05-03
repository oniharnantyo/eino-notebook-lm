package pipeline

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

type StorageStage struct {
	knowledgeRepo repositories.KnowledgeRepository
	sentenceRepo  repositories.SentenceRepository
	sourceRepo    repositories.SourceRepository
	transformer   document.Transformer
	embedder      embedding.Embedder
}

func NewStorageStage(
	knowledgeRepo repositories.KnowledgeRepository,
	sentenceRepo repositories.SentenceRepository,
	sourceRepo repositories.SourceRepository,
	transformer document.Transformer,
	embedder embedding.Embedder,
) *StorageStage {
	return &StorageStage{
		knowledgeRepo: knowledgeRepo,
		sentenceRepo:  sentenceRepo,
		sourceRepo:    sourceRepo,
		transformer:   transformer,
		embedder:      embedder,
	}
}

func (s *StorageStage) Name() string { return "StorageStage" }

func (s *StorageStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	docs, ok := input.Data.([]*schema.Document)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for StorageStage: expected []*schema.Document, got %T", input.Data)
	}

	if len(docs) == 0 {
		return StageOutput{Data: input.Data}, nil
	}

	knowledges := make([]*entities.Knowledge, len(docs))
	for i, doc := range docs {
		var firstPage, lastPage int
		if val, ok := doc.MetaData["first_page"].(int); ok {
			firstPage = val
		}
		if val, ok := doc.MetaData["last_page"].(int); ok {
			lastPage = val
		}

		headingContext, _ := doc.MetaData["heading_context"].(map[string]any)

		k, err := entities.NewKnowledge(
			input.SourceID,
			doc.Content,
			i,
			headingContext,
			firstPage,
			lastPage,
			doc.MetaData,
		)
		if err != nil {
			return StageOutput{}, fmt.Errorf("failed to create knowledge entity: %w", err)
		}
		knowledges[i] = k
	}

	// Save knowledges in batch
	if err := s.knowledgeRepo.SaveBatch(ctx, knowledges); err != nil {
		return StageOutput{}, fmt.Errorf("failed to save knowledges: %w", err)
	}

	// Process and save sentences for each chunk
	for _, chunk := range knowledges {
		sentenceDocs, err := s.transformer.Transform(ctx, []*schema.Document{
			{
				ID:      chunk.ID.String(),
				Content: chunk.Content,
			},
		})
		if err != nil {
			return StageOutput{}, fmt.Errorf("failed to split chunk %s into sentences: %w", chunk.ID, err)
		}

		if len(sentenceDocs) == 0 {
			continue
		}

		contents := make([]string, len(sentenceDocs))
		for i, doc := range sentenceDocs {
			contents[i] = doc.Content
		}

		embeddings, err := s.embedder.EmbedStrings(ctx, contents)
		if err != nil {
			return StageOutput{}, fmt.Errorf("failed to generate embeddings for sentences in chunk %s: %w", chunk.ID, err)
		}

		sentences := make([]*entities.Sentence, len(sentenceDocs))
		for i, doc := range sentenceDocs {
			sentences[i] = &entities.Sentence{
				ID:          uuid.New(),
				KnowledgeID: chunk.ID,
				Content:     doc.Content,
				Embedding:   convertToFloat32(embeddings[i]),
				Position:    i,
				Metadata: map[string]any{
					"source_id": input.SourceID.String(),
				},
				CreatedAt: time.Now(),
			}
		}

		if err := s.sentenceRepo.SaveBatch(ctx, sentences); err != nil {
			return StageOutput{}, fmt.Errorf("failed to save sentences for chunk %s: %w", chunk.ID, err)
		}
	}

	// Update source chunk count
	source, err := s.sourceRepo.GetByID(ctx, input.SourceID)
	if err != nil {
		return StageOutput{}, fmt.Errorf("failed to get source: %w", err)
	}
	if source == nil {
		return StageOutput{}, fmt.Errorf("source not found: %s", input.SourceID)
	}

	source.ChunkCount = len(knowledges)
	source.UpdatedAt = time.Now()
	if err := s.sourceRepo.Update(ctx, source); err != nil {
		return StageOutput{}, fmt.Errorf("failed to update source: %w", err)
	}

	return StageOutput{Data: knowledges}, nil
}

func convertToFloat32(v []float64) []float32 {
	res := make([]float32, len(v))
	for i, f := range v {
		res[i] = float32(f)
	}
	return res
}
