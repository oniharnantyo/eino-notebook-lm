package pipeline

import (
	"context"
	"testing"

	"github.com/oniharnantyo/eino-notebook/internal/core/application/usecases/extractor"
	"github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
	"github.com/oniharnantyo/eino-notebook/pkg/logger"
	"github.com/oniharnantyo/eino-notebook/pkg/parser/kreuzberg"
	"github.com/oniharnantyo/eino-notebook/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMetadataIntegration(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	log := logger.New(logger.LevelDebug, "text")

	// 1. Setup Data with document-level metadata
	docMetadata := map[string]any{
		"title":      "Integration Test Document",
		"authors":    []string{"Agent Gemini"},
		"page_count": 42,
		"other_key":  "should not be in source",
	}

	extractionResult := &extractor.ExtractionResult{
		Content: "Full document content",
		Chunks: []kreuzberg.KreuzbergChunk{
			{
				Content:   "Chunk 1 content",
				ChunkType: "text",
				Metadata: kreuzberg.KreuzbergChunkMeta{
					ChunkIndex: 0,
					FirstPage:  1,
					LastPage:   1,
				},
			},
		},
		Metadata: docMetadata,
	}

	data := &PipelineData{
		ExtractionResult: extractionResult,
	}

	// 2. Run KnowledgeMappingStage
	mappingStage := NewKnowledgeMappingStage(log)
	mappingOutput, err := mappingStage.Execute(ctx, StageInput{SourceID: sourceID, Data: data})
	assert.NoError(t, err)

	mappedData := mappingOutput.Data.(*PipelineData)
	assert.Len(t, mappedData.Knowledges, 1)
	
	// Verify knowledge has both chunk and doc metadata
	k0Meta := mappedData.Knowledges[0].Metadata
	assert.Equal(t, "text", k0Meta["chunk_type"])
	assert.Equal(t, "Integration Test Document", k0Meta["title"])
	assert.Equal(t, 42, k0Meta["page_count"])

	// 3. Run StorageStage
	mockKnowledgeRepo := &mockKnowledgeRepository{}
	mockSentenceRepo := &mockSentenceRepository{}
	mockImageRepo := &mockImageRepository{}
	mockSourceRepo := &mockSourceRepository{
		source: &entities.Source{
			ID:       sourceID,
			Metadata: make(map[string]any),
		},
	}

	storageStage := NewStorageStage(mockKnowledgeRepo, mockSentenceRepo, mockImageRepo, mockSourceRepo)
	_, err = storageStage.Execute(ctx, StageInput{SourceID: sourceID, Data: mappedData})
	assert.NoError(t, err)

	// 4. Verify Source Metadata
	source := mockSourceRepo.source
	assert.Equal(t, "Integration Test Document", source.Metadata["title"])
	assert.Equal(t, 42, source.Metadata["page_count"])
	assert.Equal(t, []string{"Agent Gemini"}, source.Metadata["authors"])
	
	// Verify filtered keys
	assert.NotContains(t, source.Metadata, "other_key")
	assert.NotContains(t, source.Metadata, "chunk_type")
	
	// Verify source content and size
	assert.Equal(t, "Full document content", source.Content)
	assert.Equal(t, len("Full document content"), source.TotalSize)
	assert.Equal(t, 1, source.ChunkCount)
}

func TestMetadataIntegration_EmptyMetadata(t *testing.T) {
	ctx := context.Background()
	sourceID := uuid.New()
	log := logger.New(logger.LevelDebug, "text")

	extractionResult := &extractor.ExtractionResult{
		Content: "Plain text content",
		Chunks: []kreuzberg.KreuzbergChunk{
			{
				Content:  "Chunk 1",
				Metadata: kreuzberg.KreuzbergChunkMeta{ChunkIndex: 0},
			},
		},
		Metadata: nil, // No document metadata
	}

	data := &PipelineData{
		ExtractionResult: extractionResult,
	}

	mappingStage := NewKnowledgeMappingStage(log)
	mappingOutput, err := mappingStage.Execute(ctx, StageInput{SourceID: sourceID, Data: data})
	assert.NoError(t, err)

	storageStage := NewStorageStage(&mockKnowledgeRepository{}, &mockSentenceRepository{}, &mockImageRepository{}, &mockSourceRepository{
		source: &entities.Source{ID: sourceID},
	})
	
	_, err = storageStage.Execute(ctx, StageInput{SourceID: sourceID, Data: mappingOutput.Data})
	assert.NoError(t, err)
	// Should not panic or error
}
