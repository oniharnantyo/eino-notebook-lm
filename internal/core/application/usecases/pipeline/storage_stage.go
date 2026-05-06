package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

// documentMetadataKeys defines which metadata keys from Kreuzberg should be persisted to the Source entity.
// These are document-level properties (title, authors, page_count, etc.) that describe the entire document,
// as opposed to chunk-level properties (first_page, last_page, heading_context) that belong to individual chunks.
var documentMetadataKeys = []string{
	"title", "authors", "created_by", "format_type", "pdf_version", "producer",
	"is_encrypted", "width", "height", "page_count", "output_format",
	"quality_score", "pages",
}

type StorageStage struct {
	knowledgeRepo repositories.KnowledgeRepository
	sentenceRepo  repositories.SentenceRepository
	imageRepo     repositories.ImageRepository
	sourceRepo    repositories.SourceRepository
}

func NewStorageStage(
	knowledgeRepo repositories.KnowledgeRepository,
	sentenceRepo repositories.SentenceRepository,
	imageRepo repositories.ImageRepository,
	sourceRepo repositories.SourceRepository,
) *StorageStage {
	return &StorageStage{
		knowledgeRepo: knowledgeRepo,
		sentenceRepo:  sentenceRepo,
		imageRepo:     imageRepo,
		sourceRepo:    sourceRepo,
	}
}

func (s *StorageStage) Name() string { return "StorageStage" }

func (s *StorageStage) Execute(ctx context.Context, input StageInput) (StageOutput, error) {
	data, ok := input.Data.(*PipelineData)
	if !ok {
		return StageOutput{}, fmt.Errorf("invalid input type for StorageStage: expected *PipelineData, got %T", input.Data)
	}

	if len(data.Knowledges) == 0 && len(data.Sentences) == 0 && len(data.Images) == 0 {
		return StageOutput{Data: data}, nil
	}

	// Save knowledges in batch (no embedding column populated)
	if len(data.Knowledges) > 0 {
		if err := s.knowledgeRepo.SaveBatch(ctx, data.Knowledges); err != nil {
			return StageOutput{}, fmt.Errorf("failed to save knowledges: %w", err)
		}
	}

	// Convert []Sentence intermediates to []*entities.Sentence and batch save
	if len(data.Sentences) > 0 {
		sentences := make([]*entities.Sentence, len(data.Sentences))
		for i, sent := range data.Sentences {
			sentences[i] = &entities.Sentence{
				ID:          sent.ID,
				KnowledgeID: sent.KnowledgeID,
				Content:     sent.Content,
				Embedding:   sent.Embedding,
				Position:    sent.Position,
				Metadata: map[string]any{
					"source_id": input.SourceID.String(),
				},
				CreatedAt: time.Now(),
			}
		}

		if err := s.sentenceRepo.SaveBatch(ctx, sentences); err != nil {
			return StageOutput{}, fmt.Errorf("failed to save sentences: %w", err)
		}
	}

	// Batch save images
	if len(data.Images) > 0 {
		for _, img := range data.Images {
			if err := s.imageRepo.Save(ctx, img); err != nil {
				return StageOutput{}, fmt.Errorf("failed to save image %s: %w", img.ID, err)
			}
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

	source.ChunkCount = len(data.Knowledges)

	// Merge document-level metadata into source from first knowledge chunk
	if len(data.Knowledges) > 0 {
		docMeta := extractDocumentMetadata(data.Knowledges[0].Metadata)
		if source.Metadata == nil {
			source.Metadata = make(map[string]any)
		}
		for k, v := range docMeta {
			source.Metadata[k] = v
		}
	}

	source.UpdatedAt = time.Now()
	if err := s.sourceRepo.Update(ctx, source); err != nil {
		return StageOutput{}, fmt.Errorf("failed to update source: %w", err)
	}

	return StageOutput{Data: data}, nil
}

// extractDocumentMetadata filters chunk metadata to only include document-level keys.
// Chunk-level keys (first_page, last_page, heading_context, embedding, etc.) are excluded
// to prevent them from leaking into the Source entity.
func extractDocumentMetadata(docMeta map[string]any) map[string]any {
	if docMeta == nil {
		return make(map[string]any)
	}

	result := make(map[string]any)
	for _, key := range documentMetadataKeys {
		if val, exists := docMeta[key]; exists {
			result[key] = val
		}
	}
	return result
}
